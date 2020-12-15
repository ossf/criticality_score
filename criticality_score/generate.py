# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Main python script for calculating OSS Criticality Score."""

import argparse
import csv
import sys
import time

from . import run

LANGUAGE_SEARCH_MAP = {
    'js': ['Javascript', 'Typescript'],
    'java': ['Java', 'Groovy'],
    'go': ['Go'],
    'python': ['Python'],
    'rust': ['Rust'],
}
IGNORED_KEYWORDS = ['book', 'course', 'docs', 'interview', 'tutorial']


def main():
    parser = argparse.ArgumentParser(
        description=
        'Generate a sorted criticality score list for particular language(s).')
    parser.add_argument(
        "--language",
        nargs='+',
        default=[],
        required=True,
        choices=['c', 'c++', 'go', 'js', 'java', 'rust', 'python'],
        help="List of languages to use.")
    parser.add_argument("--count",
                        type=int,
                        default=200,
                        help="Number of projects in result.")
    parser.add_argument(
        "--sample-size",
        type=int,
        default=5000,
        help="Number of projects to analyze (in descending order of stars).")

    args = parser.parse_args()

    parsed_urls = []
    for lang in args.language:
        lang = lang.lower()
        for github_lang in LANGUAGE_SEARCH_MAP.get(lang, lang):
            s = 1
            last_stars_processed = None
            while s <= args.sample_size:
                query = f'language:{github_lang} archived:false'
                if last_stars_processed:
                    # +1000 to avoid any races with star updates.
                    query += f' stars:<{last_stars_processed+1000}'
                print(f'Running query: {query}')
                g = run.get_github_auth_token()
                for repo in g.search_repositories(query=query,
                                                  sort='stars',
                                                  order='desc'):
                    repo_url = repo.html_url
                    if repo_url in parsed_urls:
                        # Github search can return duplicates, so skip if analyzed.
                        continue
                    if any(k in repo_url.lower() for k in IGNORED_KEYWORDS):
                        # Ignore uninteresting repositories.
                        continue
                    parsed_urls.append(repo_url)
                    time.sleep(0.1)
                    print(f'Found {github_lang} repository({s}): {repo_url}')
                    s += 1
                    if s > args.sample_size:
                        break
                last_stars_processed = repo.stargazers_count

    csv_writer = csv.writer(sys.stdout)
    header = None
    stats = []
    for i, repo_url in enumerate(parsed_urls):
        repo = run.get_repository(repo_url)
        output = run.get_repository_stats(repo)
        if not output:
            continue
        if not header:
            header = output.keys()
            csv_writer.writerow(header)
        csv_writer.writerow(output.values())
        stats.append(output)

    print('Result:')
    csv_writer.writerow(header)
    for i in sorted(stats, key=lambda i: i['criticality_score'],
                    reverse=True)[:args.count]:
        csv_writer.writerow(i.values())


if __name__ == "__main__":
    main()
