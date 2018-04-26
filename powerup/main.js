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
                .then(r => {
                  if (r.status >= 300) {
                    return r.text().then(text => {
                      throw new Error(text)
                    })
                  }
                  return r.json()
                }),
              token
            ]))
            .then(([card, endpoints, token]) => {
              var items = endpoints.map(end => ({
                text: `Endpoint ${end.address}`,
                callback: t => viewEndpointModal(t, card.shortLink, token, end)
              }))
              items.push({
                text: '→ Create an endpoint for incoming webhooks',
                callback: t => setupEndpointModal(t, card.shortLink, token, null)
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
                      empty: `Failed to fetch the data needed for this button: '${e.message}'`
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

function modalActions (t, cardShortLink, token, endpoint) {
  return [{
    icon: './icon.svg?color=333',
    alt: 'View',
    callback: t => viewEndpointModal(t, cardShortLink, token, endpoint),
    position: 'left'
  }, {
    icon: './icon-edit.png',
    alt: 'Edit',
    callback: t => setupEndpointModal(t, cardShortLink, token, endpoint),
    position: 'left'
  }, {
    icon: './icon-delete.png',
    alt: 'Delete',
    callback: t => fetch(
      `/trello/card?token=${token}&card=${cardShortLink}&id=${endpoint.id}`,
      {
        method: 'DELETE'
      }
    )
      .then(r => r.text().then(text => console.log('delete response', text)))
      .catch(e => console.log('error', e))
      .then(() => {
        t.set('card', 'shared', '~', Date.now())
        t.notifyParent()
        t.closeModal()
      }),
    position: 'right'
  }]
}

function setupEndpointModal (t, cardShortLink, token, endpoint) {
  return t.modal({
    url: './setup.html',
    accentColor: 'orange',
    fullscreen: false,
    title: endpoint
      ? `Editing ${endpoint.address}`
      : `Create endpoint for card ${cardShortLink}`,
    args: {
      cardShortLink,
      address: endpoint && endpoint.address,
      filter: endpoint && endpoint.filter,
      last_requests: endpoint && endpoint.last_requests
    },
    actions: endpoint ? modalActions(t, cardShortLink, token, endpoint) : []
  })
}

function viewEndpointModal (t, cardShortLink, token, endpoint) {
  return t.modal({
    url: './view.html',
    accentColor: 'orange',
    fullscreen: false,
    title: `Viewing ${endpoint.address}`,
    args: {
      cardShortLink: cardShortLink,
      address: endpoint.address,
      filter: endpoint.filter,
      last_requests: endpoint.last_requests
    },
    actions: modalActions(t, cardShortLink, token, endpoint)
  })
}
