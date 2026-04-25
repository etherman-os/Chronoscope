use crate::ConsentStatus;
use std::collections::HashMap;
use std::sync::Mutex;

static CONSENT_STORE: Mutex<Option<HashMap<String, ConsentStatus>>> = Mutex::new(None);

pub fn get_status(user_id: &str) -> ConsentStatus {
    let store = CONSENT_STORE.lock().unwrap();
    if let Some(map) = store.as_ref() {
        map.get(user_id).cloned().unwrap_or(ConsentStatus::Pending)
    } else {
        ConsentStatus::Pending
    }
}

pub fn set_status(user_id: &str, status: ConsentStatus) {
    let mut store = CONSENT_STORE.lock().unwrap();
    if store.is_none() {
        *store = Some(HashMap::new());
    }
    store.as_mut().unwrap().insert(user_id.to_string(), status);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_consent_default_pending() {
        // Reset store for test isolation
        let mut store = CONSENT_STORE.lock().unwrap();
        *store = Some(HashMap::new());
        drop(store);
        assert!(matches!(get_status("unknown_user"), ConsentStatus::Pending));
    }

    #[test]
    fn test_consent_set_and_get() {
        let mut store = CONSENT_STORE.lock().unwrap();
        *store = Some(HashMap::new());
        drop(store);

        set_status("user_1", ConsentStatus::Granted);
        assert!(matches!(get_status("user_1"), ConsentStatus::Granted));

        set_status("user_1", ConsentStatus::Denied);
        assert!(matches!(get_status("user_1"), ConsentStatus::Denied));
    }
}
