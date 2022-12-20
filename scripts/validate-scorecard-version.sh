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

# Get the path to the GitHub CLI. THis should be available in the GitHub Action Images.
GH_BIN=$(which gh)
if [ "$GH_BIN" = "" ]; then
    echo "Failed to find GitHub CLI."
    exit 1
fi

# The location of the kustomization file with the Scorecard image tags.
KUSTOMIZATION_FILE=infra/envs/base/kustomization.yaml

# The location of the go.mod file with the ossf/scorecard dependency.
GO_MOD=go.mod

KUSTOMIZATION_KEY="newTag:"
SCORECARD_REPO="ossf/scorecard"
SCORECARD_PATTERN="github.com/$SCORECARD_REPO"

# Get image tags from the kustomization file
IMAGE_TAGS=$(grep "$KUSTOMIZATION_KEY" "$KUSTOMIZATION_FILE" | sort | uniq | sed "s/^\s*$KUSTOMIZATION_KEY\s*//")

# Get the version from the go.mod file
SCORECARD_VERSION=$(grep "$SCORECARD_PATTERN" "$GO_MOD" | cut -d' ' -f2)
if [ "$SCORECARD_VERSION" = "" ]; then
    echo "Scorecard dependency missing from $GO_MOD."
    exit 1
fi

# Test 1: Ensure the same tag is used for each image.
TAG_COUNT=$(echo "$IMAGE_TAGS" | wc -l)
if [ "$TAG_COUNT" != "1" ]; then
    echo "$KUSTOMIZATION_FILE: Scorecard image tags found: $TAG_COUNT. Want: 1"
    exit 1
fi

# Test 2: Treat the Go version as a pre-release version.
SHORT_SHA=$(echo "$SCORECARD_VERSION" | sed -n 's|^.*-\([a-zA-Z0-9]*\)$|\1|p')
if [ "$SHORT_SHA" = "" ]; then
    echo "Scorecard version does not contain a SHORT_SHA"
else
    # Find the full commit sha from GitHub using the GitHub CLI
    COMMIT_SHA=$($GH_BIN api "repos/$SCORECARD_REPO/commits/$SHORT_SHA" -q .sha)
    if [ "$?" != "0" ]; then 
        : # Ignore error. Treat it like a tag.
    elif [ "$COMMIT_SHA" = "$IMAGE_TAGS" ]; then
        exit 0
    elif [ "$COMMIT_SHA" != "" ]; then
        echo "$GO_MOD $COMMIT_SHA does not match $KUSTOMIZATION_FILE $IMAGE_TAGS"
        exit 1
    fi
    # If we reach here, then $COMMIT_SHA is empty. So it is possible that the
    # $SCORECARD_VERSION is a tag.
fi

# Test 3: Treat the Go version as a Git tag - turn it into a commit ID.
TAG_URL=$($GH_BIN api "repos/$SCORECARD_REPO/git/ref/tags/$SCORECARD_VERSION" -q .object.url)
if [ "$?" != "0" ]; then
    echo "Failed to get data for $GO_MOD version $SCORECARD_VERSION"
    exit 1
fi
COMMIT_SHA=$($GH_BIN api "$TAG_URL" -q .object.sha)
if [ "$COMMIT_SHA" = "$IMAGE_TAGS" ]; then
    exit 0
elif [ "$COMMIT_SHA" != "" ]; then
    echo "$GO_MOD $COMMIT_SHA does not match $KUSTOMIZATION_FILE $IMAGE_TAGS"
    exit 1
else
    echo "No SHA found for $GO_MOD $SCORECARD_PATTERN @ $SCORECARD_VERSION"
    exit 1
fi
