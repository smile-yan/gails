# Welcome to Your New Gails3 Project!

Congratulations on generating your Gails3 application! This README will guide you through the next steps to get your project up and running.

## Getting Started

1. Navigate to your project directory in the terminal.

2. To run your application in development mode, use the following command:

   ```
   gails3 dev
   ```

   This will start your application and enable hot-reloading for both frontend and backend changes.

3. To build your application for production, use:

   ```
   gails3 build
   ```

   This will create a production-ready executable in the `build` directory.

## Exploring Gails3 Features

Now that you have your project set up, it's time to explore the features that Gails3 offers:

1. **Check out the examples**: The best way to learn is by example. Visit the `examples` directory in the `v3/examples` directory to see various sample applications.

2. **Run an example**: To run any of the examples, navigate to the example's directory and use:

   ```
   go run .
   ```

   Note: Some examples may be under development during the alpha phase.

3. **Explore the documentation**: Visit the [Gails3 documentation](https://v3.gails.io/) for in-depth guides and API references.

4. **Join the community**: Have questions or want to share your progress? Join the [Gails Discord](https://discord.gg/JDdSxwjhGf) or visit the [Gails discussions on GitHub](https://github.com/wailsapp/gails/discussions).

## Project Structure

Take a moment to familiarize yourself with your project structure:

- `frontend/`: Contains your frontend code (HTML, CSS, JavaScript/TypeScript)
- `main.go`: The entry point of your Go backend
- `app.go`: Define your application structure and methods here
- `gails.json`: Configuration file for your Gails project

## Next Steps

1. Modify the frontend in the `frontend/` directory to create your desired UI.
2. Add backend functionality in `main.go`.
3. Use `gails3 dev` to see your changes in real-time.
4. When ready, build your application with `gails3 build`.

Happy coding with Gails3! If you encounter any issues or have questions, don't hesitate to consult the documentation or reach out to the Gails community.
