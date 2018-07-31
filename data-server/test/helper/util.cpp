#include "util.h"

#include "common/ds_encoding.h"

namespace sharkstore {
namespace test {
namespace helper {

using namespace sharkstore::dataserver;

metapb::Range MakeRangeMeta(Table *t) {
    metapb::Range meta;
    meta.set_id(1);
    meta.set_start_key(std::string("\x00", 1));
    meta.set_end_key("\xff");
    meta.mutable_range_epoch()->set_version(1);
    meta.mutable_range_epoch()->set_conf_ver(1);

    meta.set_table_id(t->GetID());
    auto pks = t->GetPKs();
    if (pks.size() == 0) {
        throw std::runtime_error("invalid table(no primary key)");
    }
    for (const auto& pk : pks) {
        auto p = meta.add_primary_keys();
        p->CopyFrom(pk);
    }

    return meta;
}

static const char kKeyPrefixByte = '\x01';

void EncodeKeyPrefix(std::string *buf, uint64_t table_id) {
    buf->push_back(kKeyPrefixByte);
    EncodeUint64Ascending(buf, table_id);
}

// append encoded pk values to buf
void EncodePrimaryKey(std::string *buf, const metapb::Column& col, const std::string& val) {
    switch (col.data_type()) {
        case metapb::Tinyint:
        case metapb::Smallint:
        case metapb::Int:
        case metapb::BigInt: {
            if (!col.unsigned_()) {
                int64_t i = strtoll(val.c_str(), NULL, 10);
                EncodeVarintAscending(buf, i);
            } else {
                uint64_t i = strtoull(val.c_str(), NULL, 10);
                EncodeUvarintAscending(buf, i);
            }
            break;
        }

        case metapb::Float:
        case metapb::Double: {
            double d = strtod(val.c_str(), NULL);
            EncodeFloatAscending(buf, d);
            break;
        }

        case metapb::Varchar:
        case metapb::Binary:
        case metapb::Date:
        case metapb::TimeStamp: {
            EncodeBytesAscending(buf, val.c_str(), val.size());
            break;
        }

        default:
            throw std::runtime_error(std::string("EncodePrimaryKey: invalid column data type: ") +
                std::to_string(static_cast<int>(col.data_type())));
    }
}

void EncodeColumnValue(std::string *buf, const metapb::Column& col, const std::string& val) {
    switch (col.data_type()) {
        case metapb::Tinyint:
        case metapb::Smallint:
        case metapb::Int:
        case metapb::BigInt: {
            if (!col.unsigned_()) {
                int64_t i = strtoll(val.c_str(), NULL, 10);
                EncodeIntValue(buf, static_cast<uint32_t>(col.id()), i);
            } else {
                uint64_t i = strtoull(val.c_str(), NULL, 10);
                EncodeIntValue(buf, static_cast<uint32_t>(col.id()), static_cast<int64_t>(i));
            }
            break;
        }

        case metapb::Float:
        case metapb::Double: {
            double d = strtod(val.c_str(), NULL);
            EncodeFloatValue(buf, static_cast<uint32_t>(col.id()), d);
            break;
        }

        case metapb::Varchar:
        case metapb::Binary:
        case metapb::Date:
        case metapb::TimeStamp: {
            EncodeBytesValue(buf, static_cast<uint32_t>(col.id()), val.c_str(), val.size());
            break;
        }

        default:
            throw std::runtime_error(std::string("EncodeColumnValue: invalid column data type: ") +
                                     std::to_string(static_cast<int>(col.data_type())));
    }
}

} /* namespace helper */
} /* namespace test */
} /* namespace sharkstore */

