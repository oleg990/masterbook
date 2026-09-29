INSERT INTO users (name, email, password_hash, role)
VALUES (
    'Администратор',
    'admin@masterbook.local',
    '$2a$10$6qGl1S4mu7HKw1owAOJug.EzLrwtIiz1xv2DYGYvnb2jMqme0x1i2',
    'admin'
)
ON CONFLICT (email) DO NOTHING;
