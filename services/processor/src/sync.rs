use crate::config::Config;
use crate::indexer::TimelineEvent;
use anyhow::Result;

pub struct EventTimeline {
    pub events: Vec<TimelineEvent>,
}

pub async fn synchronize_events(
    config: &Config,
    session_id: &str,
    _video_path: &std::path::Path,
) -> Result<EventTimeline> {
    let client = config.db_pool.get().await?;
    let rows = client
        .query(
            "SELECT event_type, timestamp_ms, x, y FROM events WHERE session_id = $1::uuid ORDER BY timestamp_ms ASC",
            &[&session_id],
        )
        .await?;

    let mut events = Vec::with_capacity(rows.len());
    for row in rows {
        events.push(TimelineEvent {
            event_type: row.try_get(0).unwrap_or_default(),
            timestamp_ms: row.try_get::<_, i32>(1).unwrap_or(0) as u64,
            x: row.try_get(2).unwrap_or(0),
            y: row.try_get(3).unwrap_or(0),
        });
    }

    Ok(EventTimeline { events })
}
