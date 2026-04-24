use crate::ConsentStatus;

pub fn get_status(_user_id: &str) -> ConsentStatus {
    // In MVP: always return Granted
    ConsentStatus::Granted
}
