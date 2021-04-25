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
import logging
import math
import os
import sys
import threading
import time
import urllib

import github
import gitlab
import requests

from .constants import *  # pylint: disable=wildcard-import

logger = logging.getLogger()

_CACHED_GITHUB_TOKEN = None
_CACHED_GITHUB_TOKEN_OBJ = None

PARAMS = [
    'created_since', 'updated_since', 'contributor_count', 'org_count',
    'commit_frequency', 'recent_releases_count', 'updated_issues_count',
    'closed_issues_count', 'comment_frequency', 'dependents_count'
]


class Repository:
    """General source repository."""
    def __init__(self, repo):
        self._repo = repo
        self._last_commit = None
        self._created_since = None

    @property
    def name(self):
        raise NotImplementedError

    @property
    def url(self):
        raise NotImplementedError

    @property
    def language(self):
        raise NotImplementedError

    @property
    def last_commit(self):
        raise NotImplementedError

    @property
    def created_since(self):
        raise NotImplementedError

    @property
    def updated_since(self):
        raise NotImplementedError

    @property
    def contributor_count(self):
        raise NotImplementedError

    @property
    def org_count(self):
        raise NotImplementedError

    @property
    def commit_frequency(self):
        raise NotImplementedError

    @property
    def recent_releases_count(self):
        raise NotImplementedError

    @property
    def updated_issues_count(self):
        raise NotImplementedError

    @property
    def closed_issues_count(self):
        raise NotImplementedError

    @property
    def comment_frequency(self):
        raise NotImplementedError

    def _request_url_with_auth_headers(self, url):
        headers = {}
        if 'github.com' in url and _CACHED_GITHUB_TOKEN:
            headers = {'Authorization': f'token {_CACHED_GITHUB_TOKEN}'}

        return requests.get(url, headers=headers)

    @property
    def dependents_count(self):
        # TODO: Take package manager dependency trees into account. If we decide
        # to replace this, then find a solution for C/C++ as well.
        match = None
        parsed_url = urllib.parse.urlparse(self.url)
        repo_name = parsed_url.path.strip('/')
        dependents_url = (
            f'https://github.com/search?q="{repo_name}"&type=commits')
        for i in range(FAIL_RETRIES):
            result = self._request_url_with_auth_headers(dependents_url)
            if result.status_code == 200:
                match = DEPENDENTS_REGEX.match(result.content)
                break
            time.sleep(2**i)
        if not match:
            return 0
        return int(match.group(1).replace(b',', b''))


class GitHubRepository(Repository):
    """Source repository hosted on GitHub."""
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

    @property
    def last_commit(self):
        if self._last_commit:
            return self._last_commit
        try:
            self._last_commit = self._repo.get_commits()[0]
        except Exception:
            pass
        return self._last_commit

    def get_first_commit_time(self):
        def _parse_links(response):
            link_string = response.headers.get('Link')
            if not link_string:
                return None

            links = {}
            for part in link_string.split(','):
                match = re.match(r'<(.*)>; rel="(.*)"', part.strip())
                if match:
                    links[match.group(2)] = match.group(1)
            return links

        for i in range(FAIL_RETRIES):
            result = self._request_url_with_auth_headers(
                f'{self._repo.url}/commits')
            links = _parse_links(result)
            if links and links.get('last'):
                result = self._request_url_with_auth_headers(links['last'])
            if result.status_code == 200:
                commits = json.loads(result.content)
                if commits:
                    last_commit_time_string = (
                        commits[-1]['commit']['committer']['date'])
                    return datetime.datetime.strptime(last_commit_time_string,
                                                      "%Y-%m-%dT%H:%M:%SZ")
            time.sleep(2**i)

        return None

    # Criteria important for ranking.
    @property
    def created_since(self):
        if self._created_since:
            return self._created_since

        creation_time = self._repo.created_at

        # See if there are exist any commits before this repository creation
        # time on GitHub. If yes, then the repository creation time is not
        # correct, and it was residing somewhere else before. So, use the first
        # commit date.
        if self._repo.get_commits(until=creation_time).totalCount:
            first_commit_time = self.get_first_commit_time()
            if first_commit_time:
                creation_time = min(creation_time, first_commit_time)

        difference = datetime.datetime.utcnow() - creation_time
        self._created_since = round(difference.days / 30)
        return self._created_since

    @property
    def updated_since(self):
        last_commit_time = self.last_commit.commit.author.date
        difference = datetime.datetime.utcnow() - last_commit_time
        return round(difference.days / 30)

    @property
    def contributor_count(self):
        try:
            return self._repo.get_contributors(anon='true').totalCount
        except Exception:
            # Very large number of contributors, i.e. 5000+. Cap at 5,000.
            return 5000

    @property
    def org_count(self):
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
            return 10
        return len(orgs)

    @property
    def commit_frequency(self):
        total = 0
        for week_stat in self._repo.get_stats_commit_activity():
            total += week_stat.total
        return round(total / 52, 1)

    @property
    def recent_releases_count(self):
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
            days_since_creation = self.created_since * 30
            if not days_since_creation:
                return 0
            total_tags = 0
            try:
                total_tags = self._repo.get_tags().totalCount
            except Exception:
                # Very large number of tags, i.e. 5000+. Cap at 26.
                logger.error(f'get_tags is failed: {self._repo.url}')
                return RECENT_RELEASES_THRESHOLD
            total = round(
                (total_tags / days_since_creation) * RELEASE_LOOKBACK_DAYS)
        return total

    @property
    def updated_issues_count(self):
        issues_since_time = datetime.datetime.utcnow() - datetime.timedelta(
            days=ISSUE_LOOKBACK_DAYS)
        return self._repo.get_issues(state='all',
                                     since=issues_since_time).totalCount

    @property
    def closed_issues_count(self):
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


class GitLabRepository(Repository):
    """Source repository hosted on GitLab."""
    @staticmethod
    def _date_from_string(date_string):
        return datetime.datetime.strptime(date_string,
                                          "%Y-%m-%dT%H:%M:%S.%f%z")

    @property
    def name(self):
        return self._repo.name

    @property
    def url(self):
        return self._repo.web_url

    @property
    def language(self):
        languages = self._repo.languages()
        return (max(languages, key=languages.get)).lower()

    @property
    def last_commit(self):
        if self._last_commit:
            return self._last_commit
        self._last_commit = next(iter(self._repo.commits.list()), None)
        return self._last_commit

    @property
    def created_since(self):
        creation_time = self._date_from_string(self._repo.created_at)
        commit = None
        for commit in self._repo.commits.list(until=creation_time,
                                              as_list=False):
            pass
        if commit:
            creation_time = self._date_from_string(commit.created_at)
        difference = datetime.datetime.now(
            datetime.timezone.utc) - creation_time
        return round(difference.days / 30)

    @property
    def updated_since(self):
        difference = datetime.datetime.now(
            datetime.timezone.utc) - self._date_from_string(
                self.last_commit.created_at)
        return round(difference.days / 30)

    @property
    def contributor_count(self):
        return len(self._repo.repository_contributors(all=True))

    @property
    def org_count(self):
        # Not possible to calculate as this feature restricted to admins only.
        # https://docs.gitlab.com/ee/api/users.html#user-memberships-admin-only
        return 1

    @property
    def commit_frequency(self):
        commits_since_time = datetime.datetime.utcnow() - datetime.timedelta(
            days=365)
        iterator = self._repo.commits.list(since=commits_since_time,
                                           as_list=False)
        commits_count = sum(1 for _ in iterator)
        return round(commits_count / 52, 1)

    @property
    def recent_releases_count(self):
        count = 0
        for release in self._repo.releases.list():
            release_time = self._date_from_string(release.released_at)
            if (datetime.datetime.now(datetime.timezone.utc) -
                    release_time).days > RELEASE_LOOKBACK_DAYS:
                break
            count += 1
        count = 0
        if not count:
            for tag in self._repo.tags.list():
                tag_time = self._date_from_string(tag.commit['created_at'])
                if (datetime.datetime.now(datetime.timezone.utc) -
                        tag_time).days > RELEASE_LOOKBACK_DAYS:
                    break
                count += 1
        return count

    @property
    def updated_issues_count(self):
        issues_since_time = datetime.datetime.utcnow() - datetime.timedelta(
            days=ISSUE_LOOKBACK_DAYS)
        return self._repo.issuesstatistics.get(
            updated_after=issues_since_time).statistics['counts']['all']

    @property
    def closed_issues_count(self):
        issues_since_time = datetime.datetime.utcnow() - datetime.timedelta(
            days=ISSUE_LOOKBACK_DAYS)
        return self._repo.issuesstatistics.get(
            updated_after=issues_since_time).statistics['counts']['closed']

    @property
    def comment_frequency(self):
        comments_count = 0
        issues_since_time = datetime.datetime.utcnow() - datetime.timedelta(
            days=ISSUE_LOOKBACK_DAYS)
        for issue in self._repo.issues.list(updated_after=issues_since_time,
                                            as_list=False):
            try:
                comments_count += issue.notes.list(as_list=False).total
            except Exception:
                pass
        return round(comments_count / self.updated_issues_count, 1)


def get_param_score(param, max_value, weight=1):
    """Return paramater score given its current value, max value and
    parameter weight."""
    return (math.log(1 + param) / math.log(1 + max(param, max_value))) * weight


def get_repository_stats(repo, additional_params=None):
    """Return repository stats, including criticality score."""
    # Validate and compute additional params first.
    if not repo.last_commit:
        logger.error(f'Repo is empty: {repo.url}')
        return None
    if additional_params is None:
        additional_params = []
    additional_params_total_weight = 0
    additional_params_score = 0
    for additional_param in additional_params:
        try:
            value, weight, max_threshold = [
                int(i) for i in additional_param.split(':')
            ]
        except ValueError:
            logger.error('Parameter value in bad format: ' + additional_param)
            sys.exit(1)
        additional_params_total_weight += weight
        additional_params_score += get_param_score(value, max_threshold,
                                                   weight)

    def _worker(repo, param, return_dict):
        """worker function"""
        return_dict[param] = getattr(repo, param)

    threads = []
    return_dict = {}
    for param in PARAMS:
        thread = threading.Thread(target=_worker,
                                  args=(repo, param, return_dict))
        thread.start()
        threads.append(thread)
    for thread in threads:
        thread.join()

    # Guarantee insertion order.
    result_dict = {
        'name': repo.name,
        'url': repo.url,
        'language': repo.language,
    }
    for param in PARAMS:
        result_dict[param] = return_dict[param]

    total_weight = (CREATED_SINCE_WEIGHT + UPDATED_SINCE_WEIGHT +
                    CONTRIBUTOR_COUNT_WEIGHT + ORG_COUNT_WEIGHT +
                    COMMIT_FREQUENCY_WEIGHT + RECENT_RELEASES_WEIGHT +
                    CLOSED_ISSUES_WEIGHT + UPDATED_ISSUES_WEIGHT +
                    COMMENT_FREQUENCY_WEIGHT + DEPENDENTS_COUNT_WEIGHT +
                    additional_params_total_weight)

    criticality_score = round(
        ((get_param_score(result_dict['created_since'],
                          CREATED_SINCE_THRESHOLD, CREATED_SINCE_WEIGHT)) +
         (get_param_score(result_dict['updated_since'],
                          UPDATED_SINCE_THRESHOLD, UPDATED_SINCE_WEIGHT)) +
         (get_param_score(result_dict['contributor_count'],
                          CONTRIBUTOR_COUNT_THRESHOLD,
                          CONTRIBUTOR_COUNT_WEIGHT)) +
         (get_param_score(result_dict['org_count'], ORG_COUNT_THRESHOLD,
                          ORG_COUNT_WEIGHT)) +
         (get_param_score(result_dict['commit_frequency'],
                          COMMIT_FREQUENCY_THRESHOLD,
                          COMMIT_FREQUENCY_WEIGHT)) +
         (get_param_score(result_dict['recent_releases_count'],
                          RECENT_RELEASES_THRESHOLD, RECENT_RELEASES_WEIGHT)) +
         (get_param_score(result_dict['closed_issues_count'],
                          CLOSED_ISSUES_THRESHOLD, CLOSED_ISSUES_WEIGHT)) +
         (get_param_score(result_dict['updated_issues_count'],
                          UPDATED_ISSUES_THRESHOLD, UPDATED_ISSUES_WEIGHT)) +
         (get_param_score(
             result_dict['comment_frequency'], COMMENT_FREQUENCY_THRESHOLD,
             COMMENT_FREQUENCY_WEIGHT)) + (get_param_score(
                 result_dict['dependents_count'], DEPENDENTS_COUNT_THRESHOLD,
                 DEPENDENTS_COUNT_WEIGHT)) + additional_params_score) /
        total_weight, 5)

    # Make sure score between 0 (least-critical) and 1 (most-critical).
    criticality_score = max(min(criticality_score, 1), 0)

    result_dict['criticality_score'] = criticality_score
    return result_dict


def get_github_token_info(token_obj):
    """Return expiry information given a github token."""
    rate_limit = token_obj.get_rate_limit()
    near_expiry = rate_limit.core.remaining < 50
    wait_time = (rate_limit.core.reset - datetime.datetime.utcnow()).seconds
    return near_expiry, wait_time


def get_github_auth_token():
    """Return an un-expired github token if possible from a list of tokens."""
    global _CACHED_GITHUB_TOKEN
    global _CACHED_GITHUB_TOKEN_OBJ
    if _CACHED_GITHUB_TOKEN_OBJ:
        near_expiry, _ = get_github_token_info(_CACHED_GITHUB_TOKEN_OBJ)
        if not near_expiry:
            return _CACHED_GITHUB_TOKEN_OBJ

    github_auth_token = os.getenv('GITHUB_AUTH_TOKEN')
    assert github_auth_token, 'GITHUB_AUTH_TOKEN needs to be set.'
    tokens = github_auth_token.split(',')

    min_wait_time = None
    token_obj = None
    for token in tokens:
        token_obj = github.Github(token)
        near_expiry, wait_time = get_github_token_info(token_obj)
        if not min_wait_time or wait_time < min_wait_time:
            min_wait_time = wait_time
        if not near_expiry:
            _CACHED_GITHUB_TOKEN = token
            _CACHED_GITHUB_TOKEN_OBJ = token_obj
            return token_obj

    logger.warning(
        f'Rate limit exceeded, sleeping till reset: {round(min_wait_time / 60, 1)} minutes.'
    )
    time.sleep(min_wait_time)
    return token_obj


def get_gitlab_auth_token(host):
    """Return a gitlab token object."""
    gitlab_auth_token = os.getenv('GITLAB_AUTH_TOKEN')
    try:
        token_obj = gitlab.Gitlab(host, gitlab_auth_token)
        token_obj.auth()
    except gitlab.exceptions.GitlabAuthenticationError:
        logger.info("Auth token didn't work, trying un-authenticated. "
                    "Some params like comment_frequency will not work.")
        token_obj = gitlab.Gitlab(host)
    return token_obj


def get_repository(url):
    """Return repository object, given a url."""
    if not '://' in url:
        url = 'https://' + url

    parsed_url = urllib.parse.urlparse(url)
    repo_url = parsed_url.path.strip('/')
    if parsed_url.netloc.endswith('github.com'):
        repo = None
        try:
            repo = get_github_auth_token().get_repo(repo_url)
        except github.GithubException as exp:
            if exp.status == 404:
                return None
        return GitHubRepository(repo)
    if 'gitlab' in parsed_url.netloc:
        repo = None
        host = parsed_url.scheme + '://' + parsed_url.netloc
        token_obj = get_gitlab_auth_token(host)
        repo_url_encoded = urllib.parse.quote_plus(repo_url)
        try:
            repo = token_obj.projects.get(repo_url_encoded)
        except gitlab.exceptions.GitlabGetError as exp:
            if exp.response_code == 404:
                return None
        return GitLabRepository(repo)

    raise Exception('Unsupported url!')


def initialize_logging_handlers():
    logging.basicConfig(level=logging.INFO)
    logging.getLogger('').handlers.clear()

    console = logging.StreamHandler()
    console.setLevel(logging.INFO)
    logging.getLogger('').addHandler(console)


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

    initialize_logging_handlers()

    args = parser.parse_args()
    repo = get_repository(args.repo)
    if not repo:
        logger.error(f'Repo is not found: {args.repo}')
        return
    output = get_repository_stats(repo, args.params)
    if not output:
        return
    if args.format == 'default':
        for key, value in output.items():
            logger.info(f'{key}: {value}')
    elif args.format == 'json':
        logger.info(json.dumps(output, indent=4))
    elif args.format == 'csv':
        csv_writer = csv.writer(sys.stdout)
        csv_writer.writerow(output.keys())
        csv_writer.writerow(output.values())
    else:
        raise Exception(
            'Wrong format argument, use one of default, csv or json!')


if __name__ == "__main__":
    main()
