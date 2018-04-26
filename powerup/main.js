/* global TrelloPowerUp, fetch */

const {hasTokenStored, getToken, trelloAuth} = require('./helpers')
const Promise = TrelloPowerUp.Promise

TrelloPowerUp.initialize({
  'card-buttons': function (t) {
    return hasTokenStored(t)
      .then(yes => {
        if (yes) {
          return Promise.all([
            t.card('shortLink'),
            getToken(t)
          ])
            .then(([card, token]) => Promise.all([
              card,
              fetch(`/trello/card?token=${token}&card=${card.shortLink}`)
                .then(r => r.json())
            ]))
            .then(([card, endpoints]) => {
              var items = endpoints.map(end => ({
                text: `Endpoint ${end.address}`,
                callback: t => setupEndpointModal(t, card.shortLink, end)
              }))
              items.push({
                text: '→ Create an endpoint for incoming webhooks',
                callback: t => setupEndpointModal(t, card.shortLink, null)
              })

              return [{
                icon: './icon.svg',
                text: 'RequestHub',
                callback: t => t.popup({
                  title: 'RequestHub',
                  items
                })
              }]
            })
            .catch(e => {
              console.log('error loading card-button', e)
              return [{
                icon: './icon.svg',
                text: 'RequestHub',
                callback: t => {
                  t.popup({
                    title: 'Error fetching data.',
                    items: [],
                    search: {
                      placeholder: `Failed to fetch the data needed for this button: '${e.message}'`
                    }
                  })
                }
              }]
            })
        } else {
          return [{
            icon: './icon.svg',
            text: 'RequestHub',
            callback: t => trelloAuth(t)
          }]
        }
      })
      .catch(() => [])
  }
})

function setupEndpointModal (t, cardShortLink, endpoint) {
  return t.modal({
    url: './endpoint-setup.html',
    accentColor: 'orange',
    fullscreen: false,
    title: endpoint
      ? `${process.env.SERVICE_URL}/w/${endpoint.address}`
      : `Create endpoint for card ${cardShortLink}`,
    args: {
      cardShortLink,
      address: endpoint && endpoint.address,
      filter: endpoint && endpoint.filter
    }
  })
}
