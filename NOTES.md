## Design Notes

Emails:
- Emails cannot be retrieved from github reliably
  Users do not always have their @giantswarm.io email as their primary/public
  email which makes mapping users to slack accounts difficult.
- One idea is to enforce users to add their github account on their slack account
  this means we can then map github teams back to slack and use slack as a source
  of truth for email addresses.
- The idea here is to avoid having additional config files mapping user accounts as
  this becomes a maintainence nightmare over time.

Topics
- Teams should set the Topics field in slack to a comma separated list of interest areas.
- This will then be used to map to support technologies (e.g. support-capi, support-gitops)
- multiple teams can then share support topics by adding that technology to their team.

