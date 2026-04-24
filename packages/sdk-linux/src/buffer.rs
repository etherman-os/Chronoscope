pub struct CircularBuffer {
    storage: Vec<u8>,
    capacity: usize,
    write_pos: usize,
    read_pos: usize,
    len: usize,
}

impl CircularBuffer {
    pub fn new(capacity: usize) -> Self {
        Self {
            storage: vec![0; capacity],
            capacity,
            write_pos: 0,
            read_pos: 0,
            len: 0,
        }
    }

    pub fn write(&mut self, data: &[u8]) {
        for &byte in data {
            self.storage[self.write_pos] = byte;
            self.write_pos = (self.write_pos + 1) % self.capacity;
            if self.len < self.capacity {
                self.len += 1;
            } else {
                // Buffer is full, overwrite old data: advance read_pos
                self.read_pos = (self.read_pos + 1) % self.capacity;
            }
        }
    }

    pub fn read_chunk(&mut self) -> Option<Vec<u8>> {
        if self.len == 0 {
            return None;
        }
        let mut data = Vec::with_capacity(self.len);
        for _ in 0..self.len {
            data.push(self.storage[self.read_pos]);
            self.read_pos = (self.read_pos + 1) % self.capacity;
        }
        self.len = 0;
        Some(data)
    }

    pub fn len(&self) -> usize {
        self.len
    }

    pub fn is_empty(&self) -> bool {
        self.len == 0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_basic_write_read() {
        let mut buf = CircularBuffer::new(10);
        buf.write(b"hello");
        assert_eq!(buf.len(), 5);
        let data = buf.read_chunk().unwrap();
        assert_eq!(data, b"hello");
        assert!(buf.is_empty());
    }

    #[test]
    fn test_wrap_around() {
        let mut buf = CircularBuffer::new(5);
        buf.write(b"hello");
        buf.write(b"world");
        // Buffer should have overwritten "hello" with "world" (last 5 bytes)
        let data = buf.read_chunk().unwrap();
        assert_eq!(data, b"world");
    }
}
