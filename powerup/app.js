/* global TrelloPowerUp */

window.Promise = TrelloPowerUp.Promise
require('isomorphic-fetch')

if (location.pathname.match(/main.html/)) {
  require('./main.js')
} else if (location.pathname.match(/return-auth.html/)) {
  require('./return-auth.js')
} else if (location.pathname.match(/setup.html/)) {
  require('./setup.js')
} else if (location.pathname.match(/view.html/)) {
  require('./view.js')
}
