const path = require('path')
const axios = require('axios')
const fs = require('fs')
const {
  uploadFile,
} = require('./utils')

const GITHUB_PUBLIC_ORG = 'crawlab-team'

async function main() {
  const url = `https://api.github.com/orgs/${GITHUB_PUBLIC_ORG}/repos`
  const res = await axios.get(url)
  const pluginsData = res.data?.filter(d => d.name.match(/^crawlab-plugin-/))
  const jsonData = JSON.stringify(pluginsData)
  const filePath = 'plugins.json'
  fs.writeFileSync(filePath, jsonData)
  await uploadFile(path.resolve(filePath), filePath)
}

(async () => {
  await main()
})()
