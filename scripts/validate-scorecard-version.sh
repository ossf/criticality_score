#!/usr/bin/env bash
# Copyright 2022 Criticality Score Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Validate that the commit assigned to the Go dependency of Scorecard matches
# the production Docker images used when running Criticality Score.
#
# This check ensures that there is no version skew against Scorecard. Since
# our code depends on the behavior of various Scorecard cron libraries it is
# important that the production images built by the Scorecard project remain
# consistent.
#
# In particular the way config.yaml files are parsed, messages are passed 
# between the controller and worker, or the shard file structure, may change in
# ways that break Criticality Score if the versions are not kept in sync.

# Get the path to a Go binary. This should be available in the GitHub Action images.
GO_BIN=$(which go)
if [ "$GO_BIN" = "" ]; then
    echo "Failed to find GO binary."
    exit 1
fi

# The location of the kustomization file with the Scorecard image tags.
KUSTOMIZATION_FILE=infra/envs/base/kustomization.yaml

KUSTOMIZATION_KEY="newTag:"
SCORECARD_REPO="ossf/scorecard"
SCORECARD_IMPORT="github.com/$SCORECARD_REPO/v4"

# Get image tags from the kustomization file
IMAGE_TAGS=$(grep "$KUSTOMIZATION_KEY" "$KUSTOMIZATION_FILE" | sort | uniq | sed "s/^[[:space:]]*$KUSTOMIZATION_KEY[[:space:]]*//")

# Get the scorecard version
SCORECARD_VERSION=$($GO_BIN list -m -f '{{ .Version }}' $SCORECARD_IMPORT)
if [ "$SCORECARD_VERSION" = "" ]; then
    echo "Scorecard dependency missing."
    exit 1
fi

# Test 1: Ensure the same tag is used for each kubernetes image.
TAG_COUNT=$(echo "$IMAGE_TAGS" | wc -l)
if [ $TAG_COUNT -ne 1 ]; then
    echo "$KUSTOMIZATION_FILE: Scorecard image tags found: $TAG_COUNT. Want: 1"
    exit 1
fi

# Test 2: Extract the origin hash from the module.
COMMIT_SHA=$($GO_BIN list -m -f '{{ .Origin.Hash }}' "$SCORECARD_IMPORT@$SCORECARD_VERSION")
if [ "$?" != "0" ]; then 
    echo "Go go.mod $COMMIT_SHA does not match $KUSTOMIZATION_FILE $IMAGE_TAGS"
    exit 1
elif [ "$COMMIT_SHA" = "$IMAGE_TAGS" ]; then
    exit 0
elif [ "$COMMIT_SHA" != "" ]; then
    echo "Go go.mod $COMMIT_SHA does not match $KUSTOMIZATION_FILE $IMAGE_TAGS"
    exit 1
fi
