#ifndef CHRONOSCOPE_PRIVACY_H
#define CHRONOSCOPE_PRIVACY_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct ChronoscopePrivacyEngine ChronoscopePrivacyEngine;

ChronoscopePrivacyEngine* chronoscope_privacy_init(const char* config_json);
void chronoscope_privacy_process_frame(ChronoscopePrivacyEngine* engine, unsigned char* frame_data, unsigned int width, unsigned int height, unsigned int stride);
char* chronoscope_privacy_process_text(ChronoscopePrivacyEngine* engine, const char* text);
void chronoscope_privacy_free_string(char* s);
void chronoscope_privacy_free(ChronoscopePrivacyEngine* engine);

#ifdef __cplusplus
}
#endif

#endif // CHRONOSCOPE_PRIVACY_H
