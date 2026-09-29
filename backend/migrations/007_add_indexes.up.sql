CREATE INDEX IF NOT EXISTS idx_services_master_id ON services(master_id);
CREATE INDEX IF NOT EXISTS idx_working_hours_master_day ON working_hours(master_id, day_of_week);
CREATE INDEX IF NOT EXISTS idx_appointments_master_time ON appointments(master_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_appointments_client_time ON appointments(client_id, start_time);
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);
