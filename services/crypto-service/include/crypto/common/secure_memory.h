#pragma once

#include <vector>
#include <cstdint>
#include <cstring>
#include <span>
#include <memory>

#ifdef _WIN32
#include <windows.h>
#else
#include <sys/mman.h>
#endif

namespace crypto {

// Secure memory zeroing that won't be optimized away
inline void secure_zero(void* ptr, size_t size) {
    volatile unsigned char* p = static_cast<volatile unsigned char*>(ptr);
    while (size--) {
        *p++ = 0;
    }
}

// Constant-time comparison to prevent timing attacks
inline bool constant_time_compare(const void* a, const void* b, size_t size) {
    const volatile unsigned char* pa = static_cast<const volatile unsigned char*>(a);
    const volatile unsigned char* pb = static_cast<const volatile unsigned char*>(b);
    unsigned char result = 0;
    for (size_t i = 0; i < size; ++i) {
        result |= pa[i] ^ pb[i];
    }
    return result == 0;
}

// Custom allocator that locks memory and securely zeros on deallocation
template<typename T>
class SecureAllocator {
public:
    using value_type = T;
    using pointer = T*;
    using const_pointer = const T*;
    using reference = T&;
    using const_reference = const T&;
    using size_type = std::size_t;
    using difference_type = std::ptrdiff_t;

    template<typename U>
    struct rebind {
        using other = SecureAllocator<U>;
    };

    SecureAllocator() noexcept = default;
    
    template<typename U>
    SecureAllocator(const SecureAllocator<U>&) noexcept {}

    T* allocate(size_t n) {
        if (n > std::numeric_limits<size_t>::max() / sizeof(T)) {
            throw std::bad_alloc();
        }
        
        size_t size = n * sizeof(T);
        T* ptr = static_cast<T*>(std::malloc(size));
        
        if (!ptr) {
            throw std::bad_alloc();
        }

        // Lock memory to prevent swapping to disk
#ifdef _WIN32
        VirtualLock(ptr, size);
#else
        mlock(ptr, size);
#endif
        return ptr;
    }

    void deallocate(T* ptr, size_t n) noexcept {
        if (ptr) {
            size_t size = n * sizeof(T);
            // Securely zero memory before freeing
            secure_zero(ptr, size);
            
            // Unlock memory
#ifdef _WIN32
            VirtualUnlock(ptr, size);
#else
            munlock(ptr, size);
#endif
            std::free(ptr);
        }
    }

    template<typename U, typename... Args>
    void construct(U* ptr, Args&&... args) {
        new (ptr) U(std::forward<Args>(args)...);
    }

    template<typename U>
    void destroy(U* ptr) {
        ptr->~U();
    }
};

template<typename T, typename U>
bool operator==(const SecureAllocator<T>&, const SecureAllocator<U>&) noexcept {
    return true;
}

template<typename T, typename U>
bool operator!=(const SecureAllocator<T>&, const SecureAllocator<U>&) noexcept {
    return false;
}

// Secure vector that uses locked memory and zeros on destruction
using SecureVector = std::vector<uint8_t, SecureAllocator<uint8_t>>;
using SecureString = std::basic_string<char, std::char_traits<char>, SecureAllocator<char>>;

// RAII wrapper for secure memory
class SecureBuffer {
public:
    explicit SecureBuffer(size_t size) : data_(size) {}
    
    SecureBuffer(const uint8_t* data, size_t size) : data_(data, data + size) {}
    
    SecureBuffer(std::span<const uint8_t> data) : data_(data.begin(), data.end()) {}

    ~SecureBuffer() = default; // SecureAllocator handles zeroing

    // Non-copyable
    SecureBuffer(const SecureBuffer&) = delete;
    SecureBuffer& operator=(const SecureBuffer&) = delete;

    // Movable
    SecureBuffer(SecureBuffer&&) noexcept = default;
    SecureBuffer& operator=(SecureBuffer&&) noexcept = default;

    uint8_t* data() { return data_.data(); }
    const uint8_t* data() const { return data_.data(); }
    size_t size() const { return data_.size(); }
    bool empty() const { return data_.empty(); }

    void resize(size_t new_size) { data_.resize(new_size); }
    void clear() { 
        secure_zero(data_.data(), data_.size());
        data_.clear(); 
    }

    std::span<uint8_t> span() { return std::span<uint8_t>(data_); }
    std::span<const uint8_t> span() const { return std::span<const uint8_t>(data_); }

    // Convert to regular vector (copies data)
    std::vector<uint8_t> to_vector() const {
        return std::vector<uint8_t>(data_.begin(), data_.end());
    }

private:
    SecureVector data_;
};

// Smart pointer with secure deletion
template<typename T>
struct SecureDeleter {
    void operator()(T* ptr) const {
        if (ptr) {
            secure_zero(ptr, sizeof(T));
            delete ptr;
        }
    }
};

template<typename T>
using SecureUniquePtr = std::unique_ptr<T, SecureDeleter<T>>;

template<typename T, typename... Args>
SecureUniquePtr<T> make_secure_unique(Args&&... args) {
    return SecureUniquePtr<T>(new T(std::forward<Args>(args)...));
}

} // namespace crypto
