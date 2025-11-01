## SOP versioning.

- [ ] Story: As a floor manager, I need to be able to work on SOPs (continuous improvement) without disrupting the flow of current work being done on the floor. Let's say that there is an SOP for how to make Chair Model 01. There is an order for these chairs being worked on at the moment. The floor manager wants to work on an updated process for spraying the chairs, but the new process hasn't been tested yet. So he starts editing the SOP but rather than Publishing the updated SOP he wants to save it "as a draft".
      The user should also be able to make multiple edits before saving a draft or publishing. Ie, if the user adds 6 extra steps then saves, this should be one verion bump rather than 6.
- the /sops page should let the user view their drafts to go back to editing if they had to change tasks.
- once a manger is ready for an sop draft to be used on the floor, they can "publish" the draft, making it the most recent version.

## UI/UX Improvements

- [ ] Let's place the SOP Name, and Description above the steps when creating a new SOP.

- [ ] should be able to delete steps.
- [ ] should be able to rearrange steps.

## Web Bugs

- [ ] - updating the description
- [ ] - navigating to /sops doesn't highlight the SOP nav item in the side bar.
- [ ] when creating an SOP clicking "enter" adds a step. Clicking "Create SOP" when there's an empty step throws an error as all steps need a title. However, if it's the last step and it's blank let's just assume the user doesn't need it, remove it and submit.

Future Features

- [] PDF generation of SOPs for printing.
- [] AI Chat.
- [] AI Video Processing.
