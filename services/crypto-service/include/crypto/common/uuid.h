#pragma once

#include <string>
#include <array>
#include <cstdint>
#include <random>
#include <sstream>
#include <iomanip>

namespace crypto {

class UUID {
public:
    UUID() : bytes_{} {}

    // Generate a new UUID v4
    static UUID generate() {
        UUID uuid;
        std::random_device rd;
        std::mt19937_64 gen(rd());
        std::uniform_int_distribution<uint64_t> dis;

        uint64_t* ptr = reinterpret_cast<uint64_t*>(uuid.bytes_.data());
        ptr[0] = dis(gen);
        ptr[1] = dis(gen);

        // Set version to 4 (random)
        uuid.bytes_[6] = (uuid.bytes_[6] & 0x0F) | 0x40;
        // Set variant to RFC 4122
        uuid.bytes_[8] = (uuid.bytes_[8] & 0x3F) | 0x80;

        return uuid;
    }

    // Parse UUID from string
    static std::optional<UUID> parse(std::string_view str) {
        if (str.length() != 36) {
            return std::nullopt;
        }

        UUID uuid;
        size_t byte_idx = 0;
        
        for (size_t i = 0; i < str.length() && byte_idx < 16; ++i) {
            if (str[i] == '-') continue;
            
            if (i + 1 >= str.length()) return std::nullopt;
            
            char high = str[i];
            char low = str[i + 1];
            
            auto hex_to_int = [](char c) -> int {
                if (c >= '0' && c <= '9') return c - '0';
                if (c >= 'a' && c <= 'f') return c - 'a' + 10;
                if (c >= 'A' && c <= 'F') return c - 'A' + 10;
                return -1;
            };

            int h = hex_to_int(high);
            int l = hex_to_int(low);
            
            if (h < 0 || l < 0) return std::nullopt;
            
            uuid.bytes_[byte_idx++] = static_cast<uint8_t>((h << 4) | l);
            ++i;
        }

        return byte_idx == 16 ? std::optional<UUID>(uuid) : std::nullopt;
    }

    // Convert to string
    std::string to_string() const {
        std::ostringstream oss;
        oss << std::hex << std::setfill('0');
        
        for (size_t i = 0; i < 16; ++i) {
            if (i == 4 || i == 6 || i == 8 || i == 10) {
                oss << '-';
            }
            oss << std::setw(2) << static_cast<int>(bytes_[i]);
        }
        
        return oss.str();
    }

    bool operator==(const UUID& other) const {
        return bytes_ == other.bytes_;
    }

    bool operator!=(const UUID& other) const {
        return !(*this == other);
    }

    bool operator<(const UUID& other) const {
        return bytes_ < other.bytes_;
    }

    const std::array<uint8_t, 16>& bytes() const { return bytes_; }

private:
    std::array<uint8_t, 16> bytes_;
};

} // namespace crypto
