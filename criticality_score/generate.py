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
import logging
import os
import time

from . import run

logger = logging.getLogger()

LANGUAGE_SEARCH_MAP = {
    'c': ['C'],
    'c#': ['C#'],
    'c++': ['C++'],
    'go': ['Go'],
    'java': ['Java', 'Groovy', 'Kotlin', 'Scala'],
    'js': ['Javascript', 'Typescript', 'CoffeeScript'],
    'php': ['PHP'],
    'python': ['Python'],
    'ruby': ['Ruby'],
    'rust': ['Rust'],
    'shell': ['Shell'],
}
IGNORED_KEYWORDS = ['docs', 'interview', 'tutorial']

def get_github_repo_urls(sample_size, languages):
    urls = []
    if languages:
        for lang in languages:
            lang = lang.lower()
            for github_lang in LANGUAGE_SEARCH_MAP.get(lang, lang):
                urls = get_github_repo_urls_for_language(urls, sample_size, github_lang)
    else:
        urls = get_github_repo_urls_for_language(urls, sample_size)

    return urls

def get_github_repo_urls_for_language(urls, sample_size, github_lang=None):
    """Return repository urls given a language list and sample size."""
    samples_processed = 1

    star_min_start = 65536      # set high enough so that few repos have this many stars
    star_min = star_max = None  # min and max stars in the window to fetch
    divisor = 2                 # we start out halving the min for each successive query
    threshold = 750             # set to be significantly lower than the limit of 1000

    result_counts = []
    while True:
        query = 'archived:false'
        if github_lang:
            query += f' language:{github_lang}'

        # If we've done more than two queries, look at the last two sets of
        # results and do some extrapolation. We do this because we need to keep
        # queries small, as GitHub queries become nondeterministic and omit
        # data if there would be over 1000 results. Since we are querying by
        # star ranges, it is easy to get very large result sets as we go lower,
        # because many more repos have lower star counts than large ones.
        if len(result_counts) >= 2:
            # Stars tend to obey a power law -- there are more repositories
            # with lower numbers of stars than repositories with high numbers
            # of stars. For most languages, there are 2-4x more repos with >=
            # N/2 stars than repos with >= N stars, but this multiplier
            # decreases as N decreases. We can therefore use it to calculate a
            # pretty good upper bound on the number of repos to expect in the
            # next result set.
            multiplier = result_counts[-1] / result_counts[-2]
            expected_next_results = int(multiplier * result_counts[-1])

            logger.info(f"Last multiplier: {multiplier}")
            logger.info(f"Expecting {expected_next_results} next time\n")

            # If the expected number of results is higher than the threshold
            # beyond which GitHub may give us nondeterministic results,
            # decrease the size of the next star window so we get fewer results.
            if expected_next_results > threshold:
                # If we were at 2000 min stars and the divisior was 2, our next
                # min would've been 1000. To shrink the window, pick a divisor
                # so the next min is half as far away (i.e., 1500).
                #
                # If we start with the divisor as 2, the ith divisor is
                # (1 + 1/2**i), for i = 0, 1, 2, 3, etc.
                divisor = 2 * divisor / (divisor + 1)
                logger.info(f"That's too too many. Updated divisor to {divisor}")
                logger.info(f"Now expecting << {expected_next_results / 2}\n")

        # Construct the query for the next chunk of stars
        star_max = star_min
        if not star_max:
            star_min = star_min_start  # First time through
            query += f' stars:>={star_min}'
        else:
            star_min = int(star_min / divisor)
            query += f' stars:{star_min}..{star_max-1}'

        logger.info(f'Running query: {query}')
        token_obj = run.get_github_auth_token()
        new_result = False
        repo = None

        repo_count = 0
        for repo in token_obj.search_repositories(query=query,
                                                    sort='updated',
                                                    order='desc'):
            # Forced sleep to avoid hitting rate limit.
            time.sleep(0.1)
            repo_url = repo.html_url
            if repo_url in urls:
                # Github search can return duplicates, so skip if analyzed.
                continue
            if any(k in repo_url.lower() for k in IGNORED_KEYWORDS):
                # Ignore uninteresting repositories.
                continue
            urls.append(repo_url)
            new_result = True
            logger.info(f'Found repository'
                        f'({samples_processed}): {repo.stargazers_count} {repo_url}')
            samples_processed += 1
            repo_count += 1

        logger.info(f'\nProcessed a chunk of {repo_count} repos\n')

        # TODO: handle too-high result counts by reducing the window again.
        if repo_count > 1000:
            raise RuntimeError("Bad steering -- potentially invalid results!")

        # Sample count is a lower bound so that we are deterministic w.r.t.
        # star count. We want the top N repos by stars, along with any other
        # repos with just as many stars as the lowest star count of those N.
        # For that reason, we break outside the query loop.
        if samples_processed > sample_size:
            break

        # record how many results we are getting each time through. GitHub
        # results are nondeterministic and may have gaps if there are more than
        # 1,000 results for any query, so we try to do small queries to steer
        # away from this number.
        result_counts.append(repo_count)

    return urls

def initialize_logging_handlers(output_dir):
    log_filename = os.path.join(output_dir, 'output.log')
    logging.basicConfig(filename=log_filename, filemode='w', level=logging.INFO)

    console = logging.StreamHandler()
    console.setLevel(logging.INFO)
    logging.getLogger('').addHandler(console)

def main():
    parser = argparse.ArgumentParser(
        description=
        'Generate a sorted criticality score list for particular language(s).')
    parser.add_argument("--language",
                        nargs='+',
                        default=[],
                        required=False,
                        choices=LANGUAGE_SEARCH_MAP.keys(),
                        help="List of languages to use.")
    parser.add_argument("--output-dir",
                        type=str,
                        required=True,
                        help="Directory to place the output in.")
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

    initialize_logging_handlers(args.output_dir)

    repo_urls = get_github_repo_urls(args.sample_size, args.language)
    stats = []
    index = 1
    for repo_url in repo_urls:
        output = None
        for _ in range(3):
            try:
                repo = run.get_repository(repo_url)
                if not repo:
                    logger.error(f'Repo is not found: {repo_url}')
                    break
                output = run.get_repository_stats(repo)
                break
            except Exception as exp:
                logger.exception(f'Exception occurred when reading repo: {repo_url}\n{exp}')
        if not output:
            continue
        logger.info(f"{index} - {output['name']} - {output['url']} - {output['criticality_score']}")
        stats.append(output)
        index += 1

    if len(stats) == 0:
        return
    languages = '_'.join(args.language) if args.language else 'all'
    languages = languages.replace('+', 'plus').replace('c#', 'csharp')
    output_filename = os.path.join(args.output_dir,
                                   f'{languages}_top_{args.count}.csv')
    with open(output_filename, 'w') as file_handle:
        csv_writer = csv.writer(file_handle)
        header = output.keys()
        csv_writer.writerow(header)
        for i in sorted(stats,
                        key=lambda i: i['criticality_score'],
                        reverse=True)[:args.count]:
            csv_writer.writerow(i.values())
    logger.info(f'Wrote results: {output_filename}')


if __name__ == "__main__":
    main()
