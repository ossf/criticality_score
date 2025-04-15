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

FROM gcr.io/google.com/cloudsdktool/google-cloud-cli:slim@sha256:134f1938136304884f2d3b7365162adb1ca06d0b86c599f15a4fe107b94f7950

# Add "yq" to the image so the YAML config can be read.
RUN apt-get update -qqy && apt-get install -qqy yq

WORKDIR /bin
COPY ./infra/images/init_collect_signals/init.sh ./
RUN chmod u+x init.sh
