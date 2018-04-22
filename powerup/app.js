/* global TrelloPowerUp */

window.Promise = TrelloPowerUp.Promise
require('isomorphic-fetch')

if (location.pathname.match(/main.html/)) {
  require('./main.js')
} else if (location.pathname.match(/return-auth.html/)) {
  require('./return-auth.js')
} else if (location.pathname.match(/endpoint-setup.html/)) {
  require('./endpoint-setup.js')
}
