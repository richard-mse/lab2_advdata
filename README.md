# Laboratory 2 – Diving deeper with Neo4j
## by Richard Tyarks and Alec Schmidt

Rancher project : 25-adv-daba-tyarks-schmidt
Namespace : advdata-labo2
Neo4j Pod ID : \<Known at the end of the project>
Neo4j credentials : neo4j/testtest
Populator Pod ID : \<Known at the end of the project>
Time spend py the project : \<Known at the end of the project>


***
## Approach for data loading
At first, we decided to read the only file in a "streaming" manner. We do still believe that this would be the faster way of doing things, but as it was quite difficult and we were unable to make it work.

So the second approach was to download the file in the pod (at container run time, because at container build time was not allowed) before stripping the NumberInt() and then writing by batches. Then we did multiple runs of various batch size to select the fastest.

Because of the NumberInt() within the json file, the usual JSON librairies were unable to read the file normally. Due to a sever lack of time and motivation, we did not reimplement a JSON parser from the ground up. We decided that the time loss from reading the file, sanitizing it and writing into a new JSON file was not long enough to be an issue.

## Reading the logs

We have two pods running. The one containing Neo4j and our custom-made Populator. On the second one, the last logs before the process ends are all the various durations logs.

First is the time spent sanitizing the file.
Second is the time spent creating batchs and sending them to the Neo4j DB.
Third is the total.

All logs are in seconds.


## run Docker neo4j
```bash
docker run --name advdaba_labo2   -p7474:7474 -p7687:7687   -v $HOME/neo4j/logs:/logs   -v $HOME/neo4j/data:/data   -v $HOME/neo4j/import:/var/lib/neo4j/import   --memory="3g"   --env NEO4J_AUTH=neo4j/testtest   neo4j:latest
```

## run main.go
```bash
go run main.go
```