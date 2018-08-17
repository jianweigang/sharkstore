#include "admin_server.h"

#include "net/session.h"
#include "frame/sf_logger.h"
#include "server/range_server.h"

namespace sharkstore {
namespace dataserver {
namespace admin {

using namespace ds_adminpb;

AdminServer::AdminServer(server::ContextServer* context) :
    context_(context) {
}

AdminServer::~AdminServer() {
    Stop();
}

Status AdminServer::Start(uint16_t port) {
    net::ServerOptions sops;
    sops.io_threads_num = 0;
    sops.max_connections = 200;
    net_server_.reset(new net::Server(sops));

    auto ret = net_server_->ListenAndServe("0.0.0.0", port,
            [this](const net::Context& ctx, const net::MessagePtr& msg) {
                onMessage(ctx, msg);
            });
    if (!ret.ok()) return ret;

    FLOG_INFO("[Admin] server listen on 0.0.0.0:%u", port);

    return Status::OK();
}

Status AdminServer::Stop() {
    return Status::OK();
}

Status AdminServer::checkAuth(const ds_adminpb::AdminAuth& auth) {
    // TODO:
    return Status::OK();
}

Status AdminServer::execute(const ds_adminpb::AdminRequest& req, ds_adminpb::AdminResponse* resp) {
    switch (req.typ()) {
        case SET_CONFIG:
            return setConfig(req.set_cfg_req(), resp->mutable_set_cfg_resp());
        case GET_CONFIG:
            return getConfig(req.get_cfg_req(), resp->mutable_get_cfg_resp());
        case GET_INFO:
            return getInfo(req.get_info_req(), resp->mutable_get_info_response());
        case FORCE_SPLIT:
            return forceSplit(req.force_split_req(), resp->mutable_force_split_resp());
        case COMPACTION:
            return compaction(req.compaction_req(), resp->mutable_compaction_resp());
        case CLEAR_QUEUE:
            return clearQueue(req.clear_queue_req(), resp->mutable_clear_queue_resp());
        case GET_PENDINGS:
            return getPending(req.get_pendings_req(), resp->mutable_get_pendings_resp());
        case FLUSH_DB:
            return flushDB(req.flush_db_req(), resp->mutable_flush_db_resp());
        default:
            return Status(Status::kNotSupported, "admin type", std::to_string(req.typ()));
    }
}

void AdminServer::onMessage(const net::Context& ctx, const net::MessagePtr& msg) {
    AdminRequest req;
    if (!req.ParseFromArray(msg->body.data(), static_cast<int>(msg->body.size()))) {
        FLOG_ERROR("[Admin] deserialize failed from %s, head: %s",
                ctx.remote_addr.c_str(), msg->head.DebugString().c_str());
    }
    FLOG_INFO("[Admin] recv %s from %s.", ds_adminpb::AdminType_Name(req.typ()).c_str(), ctx.remote_addr.c_str());

    AdminResponse resp;
    Status ret = checkAuth(req.auth());
    if (ret.ok()) {
        ret = execute(req, &resp);
    }

    if (!ret.ok()) {
        FLOG_WARN("[Admin] handle %s from %s error: %s", AdminType_Name(req.typ()).c_str(),
                ctx.remote_addr.c_str(), ret.ToString().c_str());
        resp.set_code(static_cast<uint32_t>(ret.code()));
        resp.set_error_msg(ret.ToString());
    }

    auto resp_msg = net::NewMessage();
    resp_msg->head.SetFrom(msg->head);
    resp_msg->body.resize(resp.ByteSizeLong());
    resp.SerializeToArray(resp_msg->body.data(), static_cast<int>(resp_msg->body.size()));
    auto conn = ctx.session.lock();
    if (conn) {
        conn->Write(resp_msg);
    }
}

Status AdminServer::forceSplit(const ds_adminpb::ForceSplitRequest& req, ds_adminpb::ForceSplitResponse* resp) {
    auto rng = context_->range_server->Find(req.range_id());
    if (rng == nullptr) {
        return Status(Status::kNotFound, "range", std::to_string(req.range_id()));
    }
    FLOG_INFO("[Admin] force split range %" PRIu64 ", version: %" PRIu64, req.range_id(), req.version());
    auto s = rng->ForceSplit(req.version());
    if (s.code() == Status::kStaleEpoch) {
        FLOG_WARN("[Admin] force split range %" PRIu64 ", stale version: %" PRIu64,
                req.range_id(), req.version());
        return Status::OK();
    } else {
        return s;
    }
}

Status AdminServer::compaction(const ds_adminpb::CompactionRequest& req, ds_adminpb::CompactionResponse* resp) {
    auto db = context_->rocks_db;
    rocksdb::Status s;
    if (req.range_id() == 0) {
        s = db->CompactRange(rocksdb::CompactRangeOptions(), nullptr, nullptr);
    } else {
        auto rng = context_->range_server->Find(req.range_id());
        if (rng == nullptr) {
            return Status(Status::kNotFound, "range", std::to_string(req.range_id()));
        }
        auto meta = rng->options();
        resp->set_begin_key(meta.start_key());
        resp->set_end_key(meta.end_key());
        rocksdb::Slice begin = meta.start_key();
        rocksdb::Slice end = meta.end_key();
        s = db->CompactRange(rocksdb::CompactRangeOptions(), &begin, &end);
    }

    if (!s.ok()) {
        return Status(Status::kIOError, "compact range", s.ToString());
    }
    return Status::OK();
}

Status AdminServer::clearQueue(const ds_adminpb::ClearQueueRequest& req, ds_adminpb::ClearQueueResponse* resp) {
    return Status(Status::kNotSupported);
}

Status AdminServer::getPending(const ds_adminpb::GetPendingsRequest& req, ds_adminpb::GetPendingsResponse* resp) {
    return Status(Status::kNotSupported);
}

Status AdminServer::flushDB(const ds_adminpb::FlushDBRequest& req, ds_adminpb::FlushDBResponse* resp) {
    rocksdb::FlushOptions fops;
    fops.wait = req.wait();
    auto s = context_->rocks_db->Flush(fops);
    if (!s.ok()) {
        return Status(Status::kIOError, "flush", s.ToString());
    }
    return Status::OK();
}

} // namespace admin
} // namespace dataserver
} // namespace sharkstore
