/* global TrelloPowerUp */

const h = require('react-hyperscript')
const React = require('react')
const render = require('react-dom').render

const Loader = require('./loader')
const t = TrelloPowerUp.iframe()
const colors = TrelloPowerUp.util.colors

class View extends React.Component {
  constructor (props) {
    super(props)

    this.cardShortLink = t.arg('cardShortLink')
    this.address = t.arg('address')
    this.last_requests = t.arg('last_requests')
    this.state = {

    }
  }

  render () {
    if (this.state.loading) {
      return (
        h('div', [
          h('center', [
            h(Loader, {color: colors.getHexString('orange')}),
            h('p', this.state.loading)
          ])
        ])
      )
    }

    return (
      h('#view', [
        h('.address', this.address),
        h('div', [
          'To post to this endpoint, send a request to ',
          h('pre', [
            h('code', `${process.env.SERVICE_URL}/w/${this.address}`)
          ])
        ]),
        h('div', [
          h('h2', 'Last requests received'),
          h('ul', this.last_requests.length
            ? this.last_requests
            .filter(x => x)
            .map(r => this.state.viewingRequest === r
              ? h('li.active', [
                h('div', [
                  h('a', {
                    href: t.signUrl('setup.html', {
                      cardShortLink: this.cardShortlink,
                      address: this.address,
                      filter: t.arg('filter'),
                      last_requests: this.last_requests,
                      test_data: JSON.stringify(r, null, 2)
                    })
                  }, 'View in filter editor'),
                  JSON.stringify(r, null, 2)
                ])
              ])
              : h('li', {
                onClick: e => {
                  e.preventDefault()
                  this.setState({viewingRequest: r}, () => {
                    t.sizeTo('html')
                  })
                }
              }, JSON.stringify(r))
            )
            : h('p', 'No requests received on this endpoint in the last 30 days.')
          )
        ])
      ])
    )
  }
}

render(
  h(View),
  document.getElementById('root')
)
