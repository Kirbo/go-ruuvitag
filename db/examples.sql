/* Find duplicate rows */
SELECT
    "time",
    "metric",
    "tagId",
    COUNT( "tagId" )
FROM
    ruuvitag
GROUP BY
    "time", "metric", "tagId"
HAVING
    COUNT( "tagId" )> 1
ORDER BY
    "time", "metric", "tagId";

/* Delete duplicate rows */
DELETE FROM
    ruuvitag a
        USING ruuvitag b
WHERE
    a.id > b.id
    AND a."time" = b."time"
    AND a."metric" = b."metric"
    and a."tagId" = b."tagId";
