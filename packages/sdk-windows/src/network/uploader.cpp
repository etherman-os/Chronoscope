#include "network/uploader.h"

#include <windows.h>
#include <winhttp.h>
#include <string>
#include <vector>

#pragma comment(lib, "winhttp.lib")

namespace chronoscope {

ChunkUploader::ChunkUploader(const std::string& endpoint, const std::string& api_key)
    : endpoint_(endpoint), api_key_(api_key) {
}

ChunkUploader::~ChunkUploader() = default;

bool ChunkUploader::UploadChunk(const std::vector<uint8_t>& data, int index, const std::string& session_id) {
    HINTERNET hSession = WinHttpOpen(L"ChronoscopeSDK/1.0", WINHTTP_ACCESS_TYPE_DEFAULT_PROXY,
                                     WINHTTP_NO_PROXY_NAME, WINHTTP_NO_PROXY_BYPASS, 0);
    if (!hSession) return false;

    // Parse endpoint host and path (simplified: assumes http://host:port/path)
    // TODO: robust URL parsing
    std::wstring host = L"localhost";
    std::wstring path = L"/ingest";
    INTERNET_PORT port = INTERNET_DEFAULT_HTTP_PORT;

    HINTERNET hConnect = WinHttpConnect(hSession, host.c_str(), port, 0);
    if (!hConnect) {
        WinHttpCloseHandle(hSession);
        return false;
    }

    HINTERNET hRequest = WinHttpOpenRequest(hConnect, L"POST", path.c_str(), nullptr,
                                            WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES,
                                            0);
    if (!hRequest) {
        WinHttpCloseHandle(hConnect);
        WinHttpCloseHandle(hSession);
        return false;
    }

    // Build multipart/form-data body
    std::string boundary = "----ChronoscopeBoundary";
    std::string header = "Content-Type: multipart/form-data; boundary=" + boundary + "\r\n";
    std::string body;
    body += "--" + boundary + "\r\n";
    body += "Content-Disposition: form-data; name=\"session_id\"\r\n\r\n";
    body += session_id + "\r\n";
    body += "--" + boundary + "\r\n";
    body += "Content-Disposition: form-data; name=\"index\"\r\n\r\n";
    body += std::to_string(index) + "\r\n";
    body += "--" + boundary + "\r\n";
    body += "Content-Disposition: form-data; name=\"chunk\"; filename=\"chunk.bin\"\r\n";
    body += "Content-Type: application/octet-stream\r\n\r\n";
    body.append(reinterpret_cast<const char*>(data.data()), data.size());
    body += "\r\n--" + boundary + "--\r\n";

    std::wstring wheader(header.begin(), header.end());
    WinHttpAddRequestHeaders(hRequest, wheader.c_str(), static_cast<DWORD>(wheader.length()), WINHTTP_ADDREQ_FLAG_ADD);

    BOOL result = WinHttpSendRequest(hRequest, WINHTTP_NO_ADDITIONAL_HEADERS, 0,
                                     (LPVOID)body.data(), static_cast<DWORD>(body.size()),
                                     static_cast<DWORD>(body.size()), 0);
    if (result) {
        result = WinHttpReceiveResponse(hRequest, nullptr);
    }

    WinHttpCloseHandle(hRequest);
    WinHttpCloseHandle(hConnect);
    WinHttpCloseHandle(hSession);
    return result != FALSE;
}

bool ChunkUploader::Finalize(const std::string& session_id) {
    HINTERNET hSession = WinHttpOpen(L"ChronoscopeSDK/1.0", WINHTTP_ACCESS_TYPE_DEFAULT_PROXY,
                                     WINHTTP_NO_PROXY_NAME, WINHTTP_NO_PROXY_BYPASS, 0);
    if (!hSession) return false;

    std::wstring host = L"localhost";
    std::wstring path = L"/complete";
    INTERNET_PORT port = INTERNET_DEFAULT_HTTP_PORT;

    HINTERNET hConnect = WinHttpConnect(hSession, host.c_str(), port, 0);
    if (!hConnect) {
        WinHttpCloseHandle(hSession);
        return false;
    }

    HINTERNET hRequest = WinHttpOpenRequest(hConnect, L"POST", path.c_str(), nullptr,
                                            WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES,
                                            0);
    if (!hRequest) {
        WinHttpCloseHandle(hConnect);
        WinHttpCloseHandle(hSession);
        return false;
    }

    std::string body = "session_id=" + session_id;
    BOOL result = WinHttpSendRequest(hRequest, WINHTTP_NO_ADDITIONAL_HEADERS, 0,
                                     (LPVOID)body.data(), static_cast<DWORD>(body.size()),
                                     static_cast<DWORD>(body.size()), 0);
    if (result) {
        result = WinHttpReceiveResponse(hRequest, nullptr);
    }

    WinHttpCloseHandle(hRequest);
    WinHttpCloseHandle(hConnect);
    WinHttpCloseHandle(hSession);
    return result != FALSE;
}

} // namespace chronoscope
