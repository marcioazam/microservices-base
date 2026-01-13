cmake_minimum_required(VERSION 3.28)
project(crypto-service VERSION 1.0.0 LANGUAGES CXX)

# ============================================================================
# C++23 Standard (Requirement 5.1)
# ============================================================================
set(CMAKE_CXX_STANDARD 23)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# ============================================================================
# Options
# ============================================================================
option(ENABLE_SANITIZERS "Enable AddressSanitizer and UndefinedBehaviorSanitizer" OFF)
option(ENABLE_TESTING "Enable testing" ON)
option(BUILD_SHARED_LIBS "Build shared libraries" OFF)
option(ENABLE_FIPS "Enable FIPS mode for OpenSSL" OFF)

# ============================================================================
# Compiler Flags
# ============================================================================
add_compile_options(-Wall -Wextra -Wpedantic -Werror)

if(ENABLE_SANITIZERS)
    add_compile_options(-fsanitize=address,undefined -fno-omit-frame-pointer)
    add_link_options(-fsanitize=address,undefined)
endif()

# ============================================================================
# Dependencies
# ============================================================================

# OpenSSL 3.3+ (Requirement 6.1)
find_package(OpenSSL 3.3 REQUIRED)
message(STATUS "OpenSSL version: ${OPENSSL_VERSION}")

if(ENABLE_FIPS)
    add_compile_definitions(CRYPTO_FIPS_MODE=1)
    message(STATUS "FIPS mode enabled")
endif()

find_package(Protobuf REQUIRED)
find_package(gRPC CONFIG REQUIRED)
find_package(Threads REQUIRED)

# ============================================================================
# Include Directories
# ============================================================================
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/include)
include_directories(${CMAKE_CURRENT_BINARY_DIR}/gen)

# ============================================================================
# Proto Files
# ============================================================================
set(PROTO_FILES proto/crypto_service.proto)

get_target_property(grpc_cpp_plugin_location gRPC::grpc_cpp_plugin LOCATION)

foreach(proto_file ${PROTO_FILES})
    get_filename_component(proto_name ${proto_file} NAME_WE)
    set(proto_src "${CMAKE_CURRENT_BINARY_DIR}/gen/${proto_name}.pb.cc")
    set(proto_hdr "${CMAKE_CURRENT_BINARY_DIR}/gen/${proto_name}.pb.h")
    set(grpc_src "${CMAKE_CURRENT_BINARY_DIR}/gen/${proto_name}.grpc.pb.cc")
    set(grpc_hdr "${CMAKE_CURRENT_BINARY_DIR}/gen/${proto_name}.grpc.pb.h")

    add_custom_command(
        OUTPUT ${proto_src} ${proto_hdr} ${grpc_src} ${grpc_hdr}
        COMMAND ${Protobuf_PROTOC_EXECUTABLE}
        ARGS --grpc_out=${CMAKE_CURRENT_BINARY_DIR}/gen
             --cpp_out=${CMAKE_CURRENT_BINARY_DIR}/gen
             --plugin=protoc-gen-grpc=${grpc_cpp_plugin_location}
             -I${CMAKE_CURRENT_SOURCE_DIR}/proto
             ${CMAKE_CURRENT_SOURCE_DIR}/${proto_file}
        DEPENDS ${proto_file}
        COMMENT "Generating protobuf and gRPC files for ${proto_file}"
    )
    list(APPEND PROTO_SRCS ${proto_src} ${grpc_src})
    list(APPEND PROTO_HDRS ${proto_hdr} ${grpc_hdr})
endforeach()

file(MAKE_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}/gen)

# ============================================================================
# Source Files
# ============================================================================
set(CRYPTO_SOURCES
    src/crypto/common/result.cpp
    src/crypto/common/secure_memory.cpp
    src/crypto/common/uuid.cpp
    src/crypto/engine/aes_engine.cpp
    src/crypto/engine/rsa_engine.cpp
    src/crypto/engine/ecdsa_engine.cpp
    src/crypto/engine/hybrid_encryption.cpp
    src/crypto/keys/key_types.cpp
    src/crypto/keys/key_store.cpp
    src/crypto/keys/key_service.cpp
    src/crypto/clients/logging_client.cpp
    src/crypto/clients/cache_client.cpp
    src/crypto/audit/audit_logger.cpp
    src/crypto/services/encryption_service.cpp
    src/crypto/services/signature_service.cpp
    src/crypto/services/file_encryption_service.cpp
    src/crypto/auth/jwt_validator.cpp
    src/crypto/auth/rbac_engine.cpp
    src/crypto/api/grpc_server.cpp
    src/crypto/api/rest_server.cpp
    src/crypto/api/health.cpp
    src/crypto/metrics/prometheus_exporter.cpp
    src/crypto/metrics/tracing.cpp
    src/crypto/config/config_loader.cpp
)

# ============================================================================
# Library
# ============================================================================
add_library(crypto_lib ${CRYPTO_SOURCES} ${PROTO_SRCS})
target_link_libraries(crypto_lib
    PUBLIC
        OpenSSL::SSL
        OpenSSL::Crypto
        gRPC::grpc++
        protobuf::libprotobuf
        Threads::Threads
)
target_include_directories(crypto_lib PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_BINARY_DIR}/gen
)

# ============================================================================
# Main Executable
# ============================================================================
add_executable(crypto-service src/main.cpp)
target_link_libraries(crypto-service PRIVATE crypto_lib)

# ============================================================================
# Testing (Requirement 12.5, 12.6)
# ============================================================================
if(ENABLE_TESTING)
    enable_testing()
    find_package(GTest REQUIRED)
    
    # RapidCheck for property-based testing - pinned version with checksum
    include(FetchContent)
    FetchContent_Declare(
        rapidcheck
        GIT_REPOSITORY https://github.com/emil-e/rapidcheck.git
        GIT_TAG ff6af6fc683159deb51c543571c33f7282e22b76  # Pinned commit
    )
    FetchContent_MakeAvailable(rapidcheck)

    set(TEST_SOURCES
        # Unit tests - common
        tests/unit/common/result_test.cpp
        tests/unit/common/hash_utils_test.cpp
        tests/unit/common/openssl_raii_test.cpp
        # Unit tests - clients (NEW)
        tests/unit/clients/logging_client_test.cpp
        tests/unit/clients/cache_client_test.cpp
        # Unit tests - config (NEW)
        tests/unit/config/config_loader_test.cpp
        # Property tests - existing
        tests/property/aes_properties_test.cpp
        tests/property/rsa_properties_test.cpp
        tests/property/signature_properties_test.cpp
        tests/property/key_properties_test.cpp
        tests/property/file_encryption_properties_test.cpp
        tests/property/audit_properties_test.cpp
        # Property tests - NEW (2025 modernization)
        tests/property/logging_properties_test.cpp
        tests/property/cache_properties_test.cpp
        tests/property/observability_properties_test.cpp
        tests/property/config_properties_test.cpp
        tests/property/input_validation_properties_test.cpp
    )

    add_executable(crypto_tests ${TEST_SOURCES})
    target_link_libraries(crypto_tests
        PRIVATE
            crypto_lib
            GTest::gtest
            GTest::gtest_main
            rapidcheck
    )
    
    include(GoogleTest)
    gtest_discover_tests(crypto_tests)

    # Integration tests (optional, requires Docker/Testcontainers)
    option(ENABLE_INTEGRATION_TESTS "Enable integration tests with Testcontainers" OFF)
    if(ENABLE_INTEGRATION_TESTS)
        set(INTEGRATION_TEST_SOURCES
            tests/integration/logging_service_integration_test.cpp
            tests/integration/cache_service_integration_test.cpp
        )
        
        add_executable(crypto_integration_tests ${INTEGRATION_TEST_SOURCES})
        target_link_libraries(crypto_integration_tests
            PRIVATE
                crypto_lib
                GTest::gtest
                GTest::gtest_main
        )
        gtest_discover_tests(crypto_integration_tests)
    endif()
endif()

# ============================================================================
# Install
# ============================================================================
install(TARGETS crypto-service RUNTIME DESTINATION bin)
install(TARGETS crypto_lib LIBRARY DESTINATION lib ARCHIVE DESTINATION lib)
install(DIRECTORY include/ DESTINATION include)
