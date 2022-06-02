const path = require('path')
const walkSync = require('walk-sync')
const dotenv = require('dotenv')
const {
  uploadFile,
} = require('./utils')

dotenv.config()

// target directory
const targetDir = `${process.argv[2]}`

async function main() {
  // paths
  const paths = walkSync(targetDir, {
    includeBasePath: true,
    directories: false,
  })

  // iterate paths
  for (const filePath of paths) {
    const localFile = path.resolve(filePath)
    try {
      await uploadFile(localFile, filePath)
    } finally {
      // do nothing
    }
  }
}

(async () => {
  await main()
})()

