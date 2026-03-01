### High
1. On the animals card, there is the option for comment and session report with the default being comment. The default should be to add a session report and opt-out to a comment. 
2. Group admin users are missing permissions. The only option for users that Group admins should not see is the `Make Admin` as that would promote someone above their level. 
3. Update the invite email to include the username that is associate with each account

### Medium

1. The session report should include fields for rough start and end time to track the duration of the sessions
2. The comments, sessions reports and annoucements all show the username that created it. That should be changed to the first name and the first letter of the last name for now. Consider other options if there is something else that makes more sense.  Usernames are typically going to be the first letter of the first name and the last name which may not easy to follow on comments.
3. Add the ability to tag users. This would be visible on the members section. The tags will show the skill level for the memeber to be able to align with the dogs required skill level.
4. There should be the option to edit a user's username. Currently `merry`, `sophia` and `terry` don't align with the rest. Those should be updated to `mjaeger`, `twallace` and `snijem`


### Low

1. The hero image is getting lost after the container restarts. This should have been fixed in a prior PR but it seems like it's still broken
2. Develop a plan to add a schedule capability. For instance, each user would have their typical weekly schedule viewable for others. It should also have the capability for someone to request coverage for one of the scheduled shifts. This would go out as an email with a link back to the request. Whoever responds in the app would take the coverage. This needs further thought and planning.