#!/bin/bash

docker stack rm factionsbot
docker build -t factionsbot .
docker stack deploy -c ./docker-compose.yml factionsbot

