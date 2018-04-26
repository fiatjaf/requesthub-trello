/* global TrelloPowerUp, fetch, jq */

const h = require('react-hyperscript')
const React = require('react')
const render = require('react-dom').render
const debounce = require('debounce')

const Loader = require('./loader')
const {getToken} = require('./helpers')
const t = TrelloPowerUp.iframe()
const colors = TrelloPowerUp.util.colors

class Setup extends React.Component {
  constructor (props) {
    super(props)

    this.cardShortLink = t.arg('cardShortLink')
    this.address = t.arg('address')
    this.state = {
      loading: null,
      success: null,
      filter: t.arg('filter') || '.',
      testData: '{}',
      testResult: ''
    }

    this.jq = debounce(() => {
      try {
        var res = jq.raw(this.state.testData, this.state.filter)
        this.setState({
          testResult: res
        })
        t.sizeTo('html')
      } catch (e) {
        console.log('jq failed', e)
      }
    }, 700)
  }

  componentDidMount () {
    this.jq()
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

    if (this.state.success) {
      return (
        h('div', [
          h('h1', 'Endpoint saved successfully'),
          h('.address', `${process.env.SERVICE_URL}/w/${this.state.success}`),
          h('p', [
            'Now whenever data is posted to ',
            h('code', `${process.env.SERVICE_URL}/w/${this.state.success}`),
            ' it will be processed by the ',
            h('a', {target: '_blank', href: 'https://stedolan.github.io/jq/'}, 'jq'),
            ' filter ',
            h('code', this.state.filter),
            ' and a comment with the resulting text will be created on this card.'
          ])
        ])
      )
    }

    return (
      h('form#setup', {
        onSubmit: e => {
          e.preventDefault()
          this.save()
        }
      }, [
        h('label', [
          h('span', [
            h('a', {href: 'https://stedolan.github.io/jq/', target: '_blank'}, 'jq'),
            ' filter: '
          ]),
          h('input', {
            value: this.state.filter,
            onChange: e => {
              e.preventDefault()
              this.setState({filter: e.target.value})
              this.jq()
            }
          })
        ]),
        h('label', [
          h('span', 'Incoming JSON test data: '),
          h('textarea', {
            value: this.state.testData,
            onChange: e => {
              e.preventDefault()
              this.setState({testData: e.target.value})
              this.jq()
            }
          })
        ]),
        h('label', [
          h('span', 'Test result: '),
          h('pre', this.state.testResult)
        ]),
        h('button.mod-primary', 'Save')
      ])
    )
  }

  save () {
    this.setState({loading: 'Saving your endpoint settings...'})

    getToken(t)
      .then(token => fetch('/trello/card', {
        method: 'PUT',
        body: JSON.stringify({
          card: this.cardShortLink,
          address: this.address,
          filter: this.state.filter,
          token
        })
      }))
      .then(r => r.json())
      .then(address => {
        this.setState({
          loading: false,
          success: address
        })
      })
  }

  done () {
    t.closeModal()
    t.notifyParent('done')
  }
}

render(
  h(Setup),
  document.getElementById('root')
)
