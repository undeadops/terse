TABLE = terse
URL = http://localhost:8000

.PHONY: db-up
db-up:
	docker run --rm --publish 8000:8000 --name ${TABLE}-dynamodb amazon/dynamodb-local:latest -jar DynamoDBLocal.jar -sharedDb -cors "*"

.PHONY: db-create
db-create:
	aws dynamodb --endpoint-url ${URL} create-table --cli-input-json file://table.json || true
	aws dynamodb --endpoint-url ${URL} batch-write-item --request-items file://fixture.json

.PHONY: db-show
db-show:
	aws dynamodb --endpoint-url ${URL} scan --table-name ${TABLE}

.PHONY: db-del
db-del:
	aws dynamodb --endpoint-url ${URL} delete-table --table-name ${TABLE}
