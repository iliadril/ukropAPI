DROP TABLE IF EXISTS comments;

DROP INDEX IF EXISTS comments__recommendation_id__idx;

DELETE FROM permissions WHERE code = 'comments:write';