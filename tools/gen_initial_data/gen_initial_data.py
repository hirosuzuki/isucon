import pymysql
import json

connection = pymysql.connect(host='localhost',
    user='isuconapp',
    password='isunageruna',
    database='isucon',
    cursorclass=pymysql.cursors.DictCursor)

with connection:
    with connection.cursor() as cursor:
        sql = "SELECT id, title, body FROM article"
        cursor.execute(sql)
        rows = cursor.fetchall()
        articles = [
            {
                "id": row["id"],
                "data": {
                    "title": row["title"],
                    "body": row["body"].rstrip()
                }
            }
            for row in rows
        ]

print("var Set = exports.Set =", json.dumps(articles, indent=2, ensure_ascii=False))
