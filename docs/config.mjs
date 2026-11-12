const stage = process.env.NODE_ENV || "dev"
const isProduction = stage === "production"

export default {
  url: isProduction ? "https://devan.gg" : "http://localhost:4321",
  basePath: isProduction ? "/witch" : "/",
  github: "https://github.com/imdevan/witch/",
  githubDocs: "https://github.com/imdevan/witch/",
  title: "witch",
  description: "witch is which, but better. Look up shell aliases, functions, and PATH executables, see where they are defined, and see what they shadow.",
}
