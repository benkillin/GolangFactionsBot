#!/bin/bash

docker build -t factionsbot . && \
docker stack rm factionsbot && sleep 7 && \
docker stack deploy -c ./docker-compose.yml factionsbot

