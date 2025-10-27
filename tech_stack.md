Setup a project with the following infrastructure.
Docker compose.

Container #1. `nori-server`.

- backend go based server using Fiber and OPen api to create rest api's for the frontend and mobile app.
- gorm connection to the nori-db container.
  Container #2. nori-db. Postgresql
  Container #3. nori-web. svelte spa app using shadcn-svelte components.
