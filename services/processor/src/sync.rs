use crate::config::Config;
use crate::indexer::TimelineEvent;
use anyhow::Result;
use tokio_postgres::Row;

pub struct EventTimeline {
    pub events: Vec<TimelineEvent>,
}

pub async fn synchronize_events(
    config: &Config,
    session_id: &str,
    _video_path: &std::path::Path,
) -> Result<EventTimeline> {
    let session_uuid = uuid::Uuid::parse_str(session_id)
        .map_err(|e| anyhow::anyhow!("invalid session_id: {}", e))?;

    let client = config.db_pool.get().await?;
    let rows = client
        .query(
            "SELECT event_type, timestamp_ms, x, y FROM events WHERE session_id = $1::uuid ORDER BY timestamp_ms ASC",
            &[&session_uuid],
        )
        .await?;

    let mut events = Vec::with_capacity(rows.len());
    for row in rows {
        events.push(TimelineEvent {
            event_type: row.try_get(0)?,
            timestamp_ms: timestamp_ms(&row)?,
            x: row.try_get(2)?,
            y: row.try_get(3)?,
        });
    }

    Ok(EventTimeline { events })
}

fn timestamp_ms(row: &Row) -> Result<u64> {
    if let Ok(value) = row.try_get::<_, i32>(1) {
        return Ok(value.max(0) as u64);
    }
    let value = row.try_get::<_, i64>(1)?;
    Ok(value.max(0) as u64)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::indexer::TimelineEvent;

    #[test]
    fn test_event_timeline_creation() {
        let events = vec![
            TimelineEvent {
                event_type: "click".to_string(),
                timestamp_ms: 1000,
                x: 10,
                y: 20,
            },
            TimelineEvent {
                event_type: "scroll".to_string(),
                timestamp_ms: 2000,
                x: 30,
                y: 40,
            },
        ];
        let timeline = EventTimeline {
            events: events.clone(),
        };
        assert_eq!(timeline.events.len(), 2);
        assert_eq!(timeline.events[0].event_type, "click");
        assert_eq!(timeline.events[1].timestamp_ms, 2000);
    }
}
