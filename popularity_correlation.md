# Is Criticality Score a Proxy for Popularity?

In the [Hacker News Discussion](https://news.ycombinator.com/item?id=25381397) about finding Critical open source projects, certain comments [1,2,3] alleged that the criticality score assigned to a project is merely a proxy for its popularity. The assertions are not without merit; a cursory review of the [dataset](https://github.com/ossf/criticality_score#public-data) produced by the Open Source Project Criticality Score program seems to indicate that the most popular projects also have a high cricticality score. For instance, popular projects like git, mono, tensorflow, kubernetes, spark, webpack, symfony, scikit-learn, rails, rust, and oss-fuzz have high critical scores (mean criticality score is 0.9125).

We wanted to evaluate if criticality score is indeed a proxy for popularity. If the evaluation yields evidence to support this assertion, then the criticality score of a project is likely a redundant measure in the presence of the popularity of the project.

## Methodology

We evaluated the correlation between the criticality score of a repository and its popularity (quantified using GitHub Stargazers). The subsections that follow contain specifics of the evaluation methodology.

### Data

The 2,200 repositories (200 repositories in each of the 11 programming languages) with criticality scores in the [dataset](https://github.com/ossf/criticality_score#public-data) produced by the Open Source Project Criticality Score program are the subjects of study in this evaluation. We used the GitHub REST API to collect<sup>1</sup> the number of stargazers for these 2,200 repositories.

<sup>1</sup>Number of stargazers for all repositories was collected on January 20, 2021.

### Analysis

We used the Spearman's Rank Correlation Coefficient (ρ) to quantify the correlation between criticality score and number of stargazers. We used the Spearman's Rank Correlation Coefficient because the criticality score and number of stargazers were found (through the Shapiro-Wilk Test) to not follow a Normal Distribution.

## Results

The correlation between the criticality score of a repository and its popularity is shown in the Table below.

| Language   |   ρ    | Effect   |     p      |
| ---------- | -----: | -------- | ---------: |
| Rust       | 0.4176 | Moderate | 7.6612E-10 |
| Ruby       | 0.4041 | Moderate | 2.9531E-09 |
| C#         | 0.3827 | Moderate | 2.2452E-08 |
| JavaScript | 0.3682 | Moderate | 8.1631E-08 |
| Java       | 0.3378 | Moderate | 9.9907E-07 |
| C++        | 0.3213 | Moderate | 3.5030E-06 |
| PHP        | 0.2880 | Weak     | 3.5521E-05 |
| Go         | 0.2842 | Weak     | 4.5382E-05 |
| C          | 0.2552 | Weak     | 2.6567E-04 |
| Shell      | 0.2230 | Weak     | 1.5068E-03 |
| Python     | 0.1695 | Weak     | 1.6419E-02 |

> p values statistically significant at significance level (α) of 0.05.

As can be inferred from the Spearman's ρ (and the corresponding interpretation of the effect), criticality score of a repository is positively correlated with its popularity but the effect is not as strong as some of the comments [1,2,3] from the Hacker News Discussion seem to suggest.

> All statistical tests were run using `scipy` v1.6.

## Interpretation

Although some popular repositories tend to have correspondingly high criticality score, there are counter examples that warrant the need for the computation of the criticality score.

# References

[1] "The methodology is pretty silly. It rewards activity and popularity." https://news.ycombinator.com/item?id=25385795

[2] "I like this idea, which pops up here and there occasionally, but this particular "criticality score" appears to measure popularity, rather than criticality." https://news.ycombinator.com/item?id=25385562

[3] "I may have misread but the fatal error in the metric to me is that popularity of a project increases its criticality when it should decrease." https://news.ycombinator.com/item?id=25388443