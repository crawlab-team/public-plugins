const qiniu = require("qiniu");
const chalk = require("chalk");

let _config

const getConfig = () => {
  if (_config) {
    return _config
  }

  // config
  _config = new qiniu.conf.Config()

  // zone
  _config.zone = qiniu.zone[process.env.QINIU_ZONE]

  return _config
}

const uploadFile = (localFile, key) => {
  // bucket
  const bucket = process.env.QINIU_BUCKET

  // options
  const options = {
    scope: `${bucket}:${key}`,
  }

  // access key
  const accessKey = process.env.QINIU_ACCESS_KEY

  // secret key
  const secretKey = process.env.QINIU_SECRET_KEY

  // mac
  const mac = new qiniu.auth.digest.Mac(accessKey, secretKey)

  // put policy
  const putPolicy = new qiniu.rs.PutPolicy(options)

  // upload token
  const uploadToken = putPolicy.uploadToken(mac)

  // config
  const config = getConfig()

  return new Promise((resolve, reject) => {
    const formUploader = new qiniu.form_up.FormUploader(config)
    const putExtra = new qiniu.form_up.PutExtra()
    formUploader.putFile(uploadToken, key, localFile, putExtra, function (respErr,
                                                                          respBody, respInfo) {
      if (respErr) {
        throw respErr
      }
      if (respInfo.statusCode === 200) {
        console.log(`${chalk.green('uploaded')} ${localFile} => ${key}`)
        resolve()
      } else if (respInfo.statusCode === 614) {
        console.log(`${chalk.yellow('exists')} ${localFile} => ${key}`)
        resolve()
      } else {
        const errMsg = `${chalk.red('error[' + respInfo.statusCode + ']')} ${localFile} => ${key}`
        console.error(errMsg)
        reject(new Error(respBody))
      }
    })
  })
}

module.exports = {
  getConfig,
  uploadFile,
}
