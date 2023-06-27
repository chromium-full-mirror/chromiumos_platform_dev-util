# CopyBot

CopyBot is a tool that automates commit copying from a third-party
repository into Gerrit.  Currently, it's used by the Zephyr and
Coreboot projects.

[TOC]

## CopyBot vs. Copybara

Google already has very complex infrastructure to copy code from one
place to another called [Copybara].  So why does CopyBot exist?
Copybara works at the "tree level" and does not know how to
cherry-pick code between sources.  Rightfully, adding support for
cherry-picking comes with additional complexities, like figuring out
how to handle merge conflicts.

However, many of our teams need the ability to either maintain patches
on a temporary basis, or land commits in a different order to
integrate dependencies.  Thus, we need the ability to do cherry-picks,
even if it does mean extra complexity.

Should [Copybara] support this in the future in a manner which works
well for CopyBot's users, we should work to deprecate our usages of
CopyBot in favor of unified infrastructure.

[Copybara]: https://github.com/google/copybara

## CopyBot's design

CopyBot is a single Python script: `copybot.py`.

`copybot.py` is what to run to copy code from one repository to
another.  Its general usage is:

```
./copybot.py [options...] <upstream_repo>:<upstream_branch> <downstream_repo>:<downstream_branch>
```

Run `./copybot.py --help` for a complete list of options supported.

`copybot.py` will then:

1. Search for the first commit in the downstream repo which has
   `GitOrigin-RevId` or `Original-Commit-Id` specified in the footers,
   or has the exact same hash as as an upstream commit.

2. Identify the commits that need cherry-picked from that commit up to
   the current branch head (keeping in mind file filtering pattens may
   cause certain commits to be skipped).

3. Cherry-pick those commits onto the downstream repo.

4. Push the changes to Gerrit.

## Using CopyBot

CopyBot is intended to be run daily as a cron job.  The Chromium OS
deployment of CopyBot runs nightly at ~4:30 AM Mountain Time.

Your job as a downstreamer is to:

- CR+2 and CQ+2 the commits uploaded by CopyBot.

- Watch for CQ failures.

- Manually handle commits with merge conflicts (if required).  You'll
  know this happens because you'll get an email from the CI failure:
  click the link for the output, and the list of commits with merge
  conflicts will be at the bottom of the page.

- Move commits with external dependency (e.g., a Cq-Depend, or an API
  change that requires rework of other code) out of the CL stack and
  handle merging these manually, as required.  See
  [Skipping Commits](#skipping-commits) below.

### Skipping Commits

On a rare occasion, it may be necessary to skip certain commits so
they can be merged later.

To do this, apply the Gerrit hashtag `copybot-skip` to either a commit
that you've uploaded manually, or one of CopyBot's commits.  From the
Gerrit UI, you may need to click the "More" button on the left panel
to see and add the hashtags.  The only requirement is that you include
the full upstream commit hash somewhere in the commit message.

On CopyBot's next run, it will respect your wishes, and no longer
upload any commits which came from this upstream revision.  If you
change your mind in the future, just remove the hashtag.

### Triggering CopyBot Manually

You can trigger CopyBot jobs from the [LUCI Scheduler UI].

## Contributing to CopyBot

First of all, thanks!  Your improvements are very much welcomed!

Python source code should be auto-formatted by `black` and `isort`.
Run `black .` and `isort .` to do the formatting.

To run tests, use `./run_tests.sh`.

### Future Improvements

A number of improvements could be made to Copybot in the future to
make it more friendly and help keep the bus number high.  Possible
ideas include:

- Better email notifications on merge conflicts.

- Automated copying and updates of FROMPULL PRs from GitHub when
  changes are pushed.

### Tutorial Video

How to run it manually video: [copybot - manual run demo][copybot_video]
(available only to Googlers for the moment).

[LUCI Scheduler UI]: https://luci-scheduler.appspot.com/jobs/chromeos
[copybot_video]: https://drive.google.com/file/d/10aG8cMnR6TOpY5veLWSXX4MGeHA0RGel/view?resourcekey=0-zeid4OuuFAsysx0dJomQWg
