#!/bin/bash

docker stop fly-lex-shop;

docker stop fly-lex-shop-db

docker rm fly-lex-shop fly-lex-shop-db
