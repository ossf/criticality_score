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
"""setup.py for OSS Criticality Score."""
import setuptools

with open('README.md', 'r') as fh:
    long_description = fh.read()

setuptools.setup(
    name='criticality_score',
    version='1.0.7',
    author='Abhishek Arya',
    author_email='',
    description='Gives criticality score for an open source project',
    long_description=long_description,
    long_description_content_type='text/markdown',
    url='https://github.com/ossf/criticality-score',
    packages=setuptools.find_packages(),
    classifiers=[
        'Programming Language :: Python :: 3',
        'License :: OSI Approved :: Apache Software License',
        'Operating System :: OS Independent',
    ],
    install_requires=[
        'PyGithub>=1.53',
        'python-gitlab>=2.5.0',
    ],
    entry_points={
        'console_scripts': ['criticality_score=criticality_score.run:main'],
    },
    python_requires='>=3.6',
    zip_safe=False,
)
