# Laboratory 2 – Diving deeper with Neo4j
## by Richard Tyarks and Alec Schmidt



## run Docker neo4j
```bash
docker run --name advdaba_labo2   -p7474:7474 -p7687:7687   -v $HOME/neo4j/logs:/logs   -v $HOME/neo4j/data:/data   -v $HOME/neo4j/import:/var/lib/neo4j/import   --memory="3g"   --env NEO4J_AUTH=neo4j/testtest   neo4j:latest
```

## run main.go
```bash
go run main.go
```