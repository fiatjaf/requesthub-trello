var key = process.env.TRELLO_API_KEY
module.exports.trelloKey = key

module.exports.trelloAuth = trelloAuth
function trelloAuth (t) {
  return t.authorize('https://trello.com/1/authorize?expiration=never&name=RequestHub&scope=read,write&key=' + key + '&callback_method=fragment&return_url=' + location.protocol + '//' + location.host + '/powerup/return-auth.html', {
    height: 680,
    width: 580,
    validtoken: x => x
  })
    .then(token =>
      t.set('member', 'private', 'token', token)
        .then(() => token)
    )
}

module.exports.getToken = function (t) {
  return t.get('member', 'private', 'token', null)
    .then(token => token || trelloAuth(t))
}
