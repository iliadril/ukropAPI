ALTER TABLE reservations ADD CONSTRAINT reservations_date_check CHECK ( end_time > start_time );
