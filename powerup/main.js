/* global TrelloPowerUp, fetch */

const {getToken} = require('./helpers')
const Promise = TrelloPowerUp.Promise

TrelloPowerUp.initialize({
  'card-buttons': function (t) {
    return [{
      icon: './icon.svg',
      text: 'RequestHub',
      callback: t => Promise.all([
        t.card('shortLink'),
        getToken(t)
      ])
        .then(([card, token]) => Promise.all([
          card,
          fetch(`/trello/card?token=${token}&card=${card.shortLink}`)
            .then(r => r.json())
        ]))
        .then(([card, endpoints]) => {
          if (endpoints.length) {
            return [{
              icon: './icon.svg',
              text: `Endpoint: ${endpoints.length}`,
              callback: t => t.popup({
                title: 'Active endpoints for this card',
                items: endpoints.map(end => ({
                  text: `https://${process.env.SERVICE_URL}/w/${end.address}`,
                  callback: t => setupEndpointModal(t, card.shortLink, end)
                }))
              })
            }]
          } else {
            return [{
              icon: './icon.svg',
              text: 'Create webhook endpoint',
              callback: t => setupEndpointModal(t, card.shortLink, null)
            }]
          }
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
    }]
  }
})

function setupEndpointModal (t, cardShortLink, endpoint) {
  return t.modal({
    url: './endpoint-setup.html',
    accentColor: 'orange',
    fullscreen: false,
    title: endpoint
      ? `Setup endpoint ${endpoint.address}`
      : `Create endpoint for card ${cardShortLink}`,
    args: {
      cardShortLink,
      address: endpoint.address,
      filter: endpoint.filter
    }
  })
}
