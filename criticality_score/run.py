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
import datetime
import json
import math
import os
import sys
import time
import urllib

import github
import requests

from .constants import *


class Repository:
    def __init__(self, repo):
        self._repo = repo


class GitHubRepository(Repository):
    # General metadata attributes.
    @property
    def name(self):
        return self._repo.name

    @property
    def url(self):
        return self._repo.html_url

    @property
    def language(self):
        return self._repo.language

    # Criteria important for ranking.
    @property
    def created_since(self):
        difference = datetime.datetime.utcnow() - self._repo.created_at
        return round(difference.days / 30)

    @property
    def updated_since(self):
        last_commit = self._repo.get_commits()[0]
        last_commit_time = last_commit.commit.author.date
        difference = datetime.datetime.utcnow() - last_commit_time
        return round(difference.days / 30)

    @property
    def contributors(self):
        try:
            return self._repo.get_contributors(anon='true').totalCount
        except Exception:
            # Very large number of contributors, i.e. 5000+. Cap at 5,000.
            return 5000

    def get_contributor_orgs(self):
        def _filter_name(org_name):
            return org_name.lower().replace('inc.', '').replace(
                'llc', '').replace('@', '').replace(' ', '').rstrip(',')

        orgs = set()
        contributors = self._repo.get_contributors()[:TOP_CONTRIBUTOR_COUNT]
        try:
            for contributor in contributors:
                if contributor.company:
                    orgs.add(_filter_name(contributor.company))
        except Exception:
            # Very large number of contributors, i.e. 5000+. Cap at 10.
            return [None] * 10
        return sorted(orgs)

    @property
    def commit_frequency(self):
        total = 0
        for week_stat in self._repo.get_stats_commit_activity():
            total += week_stat.total
        return round(total / 52, 1)

    @property
    def recent_releases(self):
        total = 0
        for release in self._repo.get_releases():
            if (datetime.datetime.utcnow() -
                    release.created_at).days > RELEASE_LOOKBACK_DAYS:
                continue
            total += 1
        if not total:
            # Make rough estimation of tags used in last year from overall
            # project history. This query is extremely expensive, so instead
            # do the rough calculation.
            days_since_creation = (datetime.datetime.utcnow() -
                                   self._repo.created_at).days
            if not days_since_creation:
                return 0
            total_tags = self._repo.get_tags().totalCount
            total = round(
                (total_tags / days_since_creation) * RELEASE_LOOKBACK_DAYS)
        return total

    @property
    def updated_issues(self):
        issues_since_time = datetime.datetime.utcnow() - datetime.timedelta(
            days=ISSUE_LOOKBACK_DAYS)
        return self._repo.get_issues(state='all',
                                     since=issues_since_time).totalCount

    @property
    def closed_issues(self):
        issues_since_time = datetime.datetime.utcnow() - datetime.timedelta(
            days=ISSUE_LOOKBACK_DAYS)
        return self._repo.get_issues(state='closed',
                                     since=issues_since_time).totalCount

    @property
    def comment_frequency(self):
        issues_since_time = datetime.datetime.utcnow() - datetime.timedelta(
            days=ISSUE_LOOKBACK_DAYS)
        issue_count = self._repo.get_issues(state='all',
                                            since=issues_since_time).totalCount
        if not issue_count:
            return 0
        comment_count = self._repo.get_issues_comments(
            since=issues_since_time).totalCount
        return round(comment_count / issue_count, 1)

    def get_dependents(self):
        repo_name = self.url.replace('https://github.com/', '')
        dependents_url = (
            f'https://github.com/search?q="{repo_name}"&type=commits')
        content = b''
        for _ in range(3):
            result = requests.get(dependents_url)
            if result.status_code == 200:
                content = result.content
                break
            time.sleep(10)
        match = DEPENDENTS_REGEX.match(content)
        if not match:
            return 0
        return int(match.group(1).replace(b',', b''))


def get_param_score(param, max_value, weight=1):
    """Return paramater score given its current value, max value and
    parameter weight."""
    return (math.log(1 + param) / math.log(1 + max(param, max_value))) * weight


def get_repository_stats(repo, additional_params=[]):
    """Return repository stats, including criticality score."""
    # Validate and compute additional params first.
    additional_params_total_weight = 0
    additional_params_score = 0
    for additional_param in additional_params:
        try:
            value, weight, max_threshold = [
                int(i) for i in additional_param.split(':')
            ]
        except ValueError:
            print('Parameter value in bad format: ' + additional_param,
                  file=sys.stderr)
            sys.exit(1)
        additional_params_total_weight += weight
        additional_params_score += get_param_score(value, max_threshold,
                                                   weight)

    created_since = repo.created_since
    updated_since = repo.updated_since
    contributor_count = repo.contributors
    org_count = len(repo.get_contributor_orgs())
    commit_frequency = repo.commit_frequency
    recent_releases_count = repo.recent_releases
    updated_issues_count = repo.updated_issues
    closed_issues_count = repo.closed_issues
    comment_frequency = repo.comment_frequency
    dependents_count = repo.get_dependents()

    total_weight = (CREATED_SINCE_WEIGHT + UPDATED_SINCE_WEIGHT +
                    CONTRIBUTOR_COUNT_WEIGHT + ORG_COUNT_WEIGHT +
                    COMMIT_FREQUENCY_WEIGHT + RECENT_RELEASES_WEIGHT +
                    CLOSED_ISSUES_WEIGHT + UPDATED_ISSUES_WEIGHT +
                    COMMENT_FREQUENCY_WEIGHT + DEPENDENTS_COUNT_WEIGHT +
                    additional_params_total_weight)

    criticality_score = round(
        (get_param_score(created_since, CREATED_SINCE_THRESHOLD,
                         CREATED_SINCE_WEIGHT) +
         get_param_score(updated_since, UPDATED_SINCE_THRESHOLD,
                         UPDATED_SINCE_WEIGHT) +
         get_param_score(contributor_count, CONTRIBUTOR_COUNT_THRESHOLD,
                         CONTRIBUTOR_COUNT_WEIGHT) +
         get_param_score(org_count, ORG_COUNT_THRESHOLD, ORG_COUNT_WEIGHT) +
         get_param_score(commit_frequency, COMMIT_FREQUENCY_THRESHOLD,
                         COMMIT_FREQUENCY_WEIGHT) +
         get_param_score(recent_releases_count, RECENT_RELEASES_THRESHOLD,
                         RECENT_RELEASES_WEIGHT) +
         get_param_score(closed_issues_count, CLOSED_ISSUES_THRESHOLD,
                         CLOSED_ISSUES_WEIGHT) +
         get_param_score(updated_issues_count, UPDATED_ISSUES_THRESHOLD,
                         UPDATED_ISSUES_WEIGHT) +
         get_param_score(comment_frequency, COMMENT_FREQUENCY_THRESHOLD,
                         COMMENT_FREQUENCY_WEIGHT) +
         get_param_score(dependents_count, DEPENDENTS_COUNT_THRESHOLD,
                         DEPENDENTS_COUNT_WEIGHT) + additional_params_score) /
        total_weight, 5)

    return {
        'name': repo.name,
        'url': repo.url,
        'language': repo.language,
        'created_since': created_since,
        'updated_since': updated_since,
        'contributor_count': contributor_count,
        'org_count': org_count,
        'commit_frequency': commit_frequency,
        'recent_releases_count': recent_releases_count,
        'closed_issues_count': closed_issues_count,
        'updated_issues_count': updated_issues_count,
        'comment_frequency': comment_frequency,
        'dependents_count': dependents_count,
        'criticality_score': criticality_score,
    }


def get_github_token_info(g):
    """Return expiry information given a github token."""
    rate_limit = g.get_rate_limit()
    near_expiry = rate_limit.core.remaining < 50
    wait_time = (rate_limit.core.reset - datetime.datetime.utcnow()).seconds
    return near_expiry, wait_time


_cached_github_token = None


def get_github_auth_token():
    """Return an un-expired github token if possible from a list of tokens."""
    global _cached_github_token
    if _cached_github_token:
        near_expiry, _ = get_github_token_info(_cached_github_token)
        if not near_expiry:
            return _cached_github_token

    github_auth_token = os.getenv('GITHUB_AUTH_TOKEN')
    assert github_auth_token, 'GITHUB_AUTH_TOKEN needs to be set.'
    tokens = github_auth_token.split(',')
    wait_time = None
    g = None
    for i, token in enumerate(tokens):
        g = github.Github(token)
        near_expiry, wait_time = get_github_token_info(g)
        if not near_expiry:
            _cached_github_token = g
            return g
    print(f'Rate limit exceeded, sleeping till reset: {wait_time} seconds.',
          file=sys.stderr)
    time.sleep(wait_time)
    return g


def get_repository(url):
    """Return repository object, given a url."""
    if not '://' in url:
        url = 'https://' + url

    parsed_url = urllib.parse.urlparse(url)
    if parsed_url.netloc.endswith('github.com'):
        g = get_github_auth_token()
        repo_url = parsed_url.path.strip('/')
        repo = GitHubRepository(g.get_repo(repo_url))
        return repo

    raise Exception('Unsupported url!')


def main():
    parser = argparse.ArgumentParser(
        description='Gives criticality score for an open source project')
    parser.add_argument("--repo",
                        type=str,
                        required=True,
                        help="repository url")
    parser.add_argument(
        "--format",
        type=str,
        default='default',
        choices=['default', 'csv', 'json'],
        help="output format. allowed values are [default, csv, json]")
    parser.add_argument(
        '--params',
        nargs='+',
        default=[],
        help='Additional parameters in form <value>:<weight>:<max_threshold>',
        required=False)

    args = parser.parse_args()
    r = get_repository(args.repo)
    output = get_repository_stats(r, args.params)
    if args.format == 'default':
        for key, value in output.items():
            print(f'{key}: {value}')
    elif args.format == 'json':
        print(json.dumps(output, indent=4))
    elif args.format == 'csv':
        csv_writer = csv.writer(sys.stdout)
        csv_writer.writerow(output.keys())
        csv_writer.writerow(output.values())
    else:
        raise Exception(
            'Wrong format argument, use one of default, csv or json!')


if __name__ == "__main__":
    main()
