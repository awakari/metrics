#!/bin/bash

export REGISTRY=ghcr.io
export ORG=awakari
export COMPONENT=metrics
export SLUG=${REGISTRY}/${ORG}/${COMPONENT}
export VERSION=$(git describe --tags --abbrev=0 | cut -c 2-)
echo "Releasing version: $VERSION"
docker tag ${ORG}/${COMPONENT} "${SLUG}":"${VERSION}"
docker push "${SLUG}":"${VERSION}"
