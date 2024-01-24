#!/bin/sh
# Copyright 2023 Criticality Score Authors
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

# Test usage (from the base dir of the repo):
# docker build . -f containers/init_collect_signals/Dockerfile -t criticality_score_init_collection
# docker run
#    -v /tmp:/output \
#    -v $HOME/.config/gcloud:/root/.config/gcloud \
#    -v $HOME/path/to/config.yaml:/etc/config.yaml \
#    -ti criticality_score_init_collection \
#    /bin/init.sh /etc/config.yaml

CONFIG_FILE="$1"

# Read the appropriate settings from the YAML config file.
BUCKET_URL=`yq -r '."additional-params"."input-bucket".url' "$CONFIG_FILE"`
BUCKET_PREFIX_FILE=`yq -r '."additional-params"."input-bucket"."prefix-file"' "$CONFIG_FILE"`
OUTPUT_FILE=`yq -r '."additional-params".criticality."local-url-data-file"' "$CONFIG_FILE"`
echo "bucket url = $BUCKET_URL"
echo "bucket prefix file = $BUCKET_PREFIX_FILE"
echo "url data file = $OUTPUT_FILE"

# Exit early if $OUTPUT_FILE already exists. This ensures $OUTPUT_FILE remains
# stable across any restart. Since the file is created atomically we can be
# confident that the file won't be corrupt or truncated.
if [ -f "$OUTPUT_FILE" ]; then
    echo "Exiting early. $OUTPUT_FILE already exists."
    exit 0
fi

LATEST_PREFIX=`gsutil cat "$BUCKET_URL"/"$BUCKET_PREFIX_FILE"`
echo "latest prefix = $LATEST_PREFIX"

# Deinfe some temporary files based on OUTPUT_FILE so they're on the same volume.
TMP_OUTPUT_FILE_1="$OUTPUT_FILE-tmp-1"
TMP_OUTPUT_FILE_2="$OUTPUT_FILE-tmp-2"

# Iterate through all the files to merge all together.
touch "$TMP_OUTPUT_FILE_1"
for file in `gsutil ls "$BUCKET_URL"/"$LATEST_PREFIX"`; do
    echo "reading $file"
    # Read the file, remove the header and turn it into a plain list of repos.
    gsutil cat "$file" | tail -n +2 | cut -d',' -f1 >> "$TMP_OUTPUT_FILE_1"
done

# Ensure the file contains only one entry per repo, and shuffle it.
sort "$TMP_OUTPUT_FILE_1" | uniq | shuf > "$TMP_OUTPUT_FILE_2"
rm "$TMP_OUTPUT_FILE_1"

# Move the final tmp file to the output file to ensure the change is atomic.
mv "$TMP_OUTPUT_FILE_2" "$OUTPUT_FILE"

echo "wrote $OUTPUT_FILE"
