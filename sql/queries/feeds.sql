-- name: Feeds :many

SELECT feeds.name, feeds.url, users.name AS user_name
FROM feeds
JOIN users 
ON users.id = feeds.user_id;
