#!/bin/bash

export SLUG=ghcr.io/awakari/metrics
export VERSION=latest
docker tag awakari/metrics "${SLUG}":"${VERSION}"
docker push "${SLUG}":"${VERSION}"
