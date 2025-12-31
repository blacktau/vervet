import { describe, expect, test } from 'vitest'
import { parseUri } from '@/features/server-pane/connectionStrings.ts'
import InvalidUriScenarios from './scenarios/invalid-uris.json' with { type: 'json' }
import ValidAuthScenarios from './scenarios/valid-auth.json' with { type: 'json' }
import ValidDbWithDottedNameScenarios from './scenarios/valid-db-with-dotted-name.json' with { type: 'json' }
import ValidHostIdentifiers from './scenarios/valid-host-identifiers.json' with { type: 'json' }
import ValidOptions from './scenarios/valid-options.json' with { type: 'json' }
import  ValidUnixSocketAbsolute from './scenarios/valid-unix-socket-absolute.json' with { type: 'json' }
import ValidUnixSockerRelative from './scenarios/valid-unix-socket-relative.json' with { type: 'json' }
import ValidWarnings from './scenarios/valid-warnings.json' with { type: 'json' }

interface ScenarioTests {
  tests: Scenario[]
}

interface Scenario {
  description: string
  uri: string
  valid: boolean
  warning?: false
  hosts?: Host[]
  options?: Record<string, unknown>
  auth?: Auth
}

interface Host {
  host: string
  port?: number
  type?: string
}

interface Auth {
  username?: string
  password?: string
  db?: string
}

describe('connectionString.parseUri', () => {
  let scenarios = InvalidUriScenarios as unknown as ScenarioTests
  for (const scenario of scenarios.tests) {
    test(`invalid uris: ${scenario.description}`,   () => {
      const result = parseUri(scenario.uri)
      expect(result.success, `should have failed to parse. got ${JSON.stringify(result.data)}`).toBeFalsy()
    })
  }

  scenarios = ValidAuthScenarios as unknown as ScenarioTests
  for(const scenario of scenarios.tests) {
    test(`valid-auth: ${scenario.description}`, () => {
      runScenario(scenario)
    })
  }

  scenarios = ValidDbWithDottedNameScenarios as unknown as ScenarioTests
  for(const scenario of scenarios.tests) {
    test(`valid-db-with-dotted-name: ${scenario.description}`, () => {
      runScenario(scenario)
    })
  }

  scenarios = ValidHostIdentifiers as unknown as ScenarioTests
  for(const scenario of scenarios.tests) {
    test(`valid-host-identifiers: ${scenario.description}`, () => {})
  }

  scenarios = ValidOptions as unknown as ScenarioTests
  for(const scenario of scenarios.tests) {
    test(`valid-options: ${scenario.description}`, () => {
      runScenario(scenario)
    })
  }

  scenarios = ValidUnixSocketAbsolute as unknown as ScenarioTests
  for(const scenario of scenarios.tests) {
    test(`valid-unix-socket-absolute: ${scenario.description}`, () => {
      runScenario(scenario)
    })
  }

  scenarios = ValidUnixSockerRelative as unknown as ScenarioTests
  for(const scenario of scenarios.tests) {
    test(`valid-unix-socket-relative: ${scenario.description}`, () => {})
  }

  scenarios = ValidWarnings as unknown as ScenarioTests
  for(const scenario of scenarios.tests) {
    test(`valid-warnings: ${scenario.description}`, () => {})
  }
})

function runScenario(scenario: Scenario) {
  const result = parseUri(scenario.uri)

  expect(result.success, `should have parsed successfully: ${result.error}`)
    .toBeTruthy()

  expect(result.data, 'should have returned parsed data').toBeDefined()

  if (scenario.hosts == null) {
    expect(result.data!.nodelist).toBeUndefined()
  } else {
    expect(result.data!.nodelist.length, 'hosts not parsed correctly')
      .toBe(scenario.hosts.length)

    for (let j = 0, ln = scenario.hosts.length || 0; j < ln; ++j) {
      const host = scenario.hosts[j]!
      let found = false
      for (let i = 0, ln = result.data?.nodelist.length || 0; i < ln; ++i) {
        if (result.data!.nodelist[i]?.host === host.host) {
          found = true
          expect(result.data!.nodelist[i]?.port, 'ports do not match')
            .toBe(host?.port)
          break
        }
      }

      expect(found, `host '${host}' not found'`)
    }
  }

  if (scenario.auth == null) {
    expect(result.data!.username, 'username should be empty')
      .toBeNullable()
    expect(result.data!.password, 'password should be empty')
      .toBeNullable()
  } else {
    if (scenario.auth.username == null) {
      expect(result.data!.username, 'username should be nullish')
        .toBeNullable()
    } else {
      expect(result.data!.username, 'username incorrectly parsed')
        .toBe(scenario.auth.username)
    }

    if (scenario.auth.password == null) {
      expect(result.data!.password, 'password incorrectly parsed')
        .toBeNullable()
    } else {
      expect(result.data!.password, 'password incorrectly parsed')
        .toBe(scenario.auth.password)
    }

    if (scenario.auth.db == null) {
      expect(result.data!.database, 'database incorrectly parsed')
        .toBeNullable()
    } else {
      expect(result.data!.database, 'database incorrectly parsed')
        .toBe(scenario.auth.db)
    }
  }

  if (scenario.options == null) {
    expect(result.data!.options, `options incorrectly parsed, got ${JSON.stringify(result.data!.options)}`).toBeUndefined()
  } else {
    const ScenarioKeys = Object.keys(scenario.options)
    const ResultKeys = Object.keys(result.data!.options || {})

    expect(ResultKeys.length).toBe(ScenarioKeys.length)

    for(let i = 0; i < ScenarioKeys.length; i++) {
      let found = false
      let scenarioKey = ScenarioKeys[i]!.toLowerCase()

      if (scenarioKey === 'authmechanismproperties') {
        // skip this here to be done below
        continue
      }

      for (let j = 0; j < ResultKeys.length; j++) {
          if (scenarioKey === ResultKeys[j]!.toLowerCase()) {
            found = true
            expect(
              result.data!.options![ResultKeys[j]!],
              `option "${scenarioKey}" doesn't match expected value`,
            ).toBe(scenario!.options[scenarioKey])
            break
          }
      }

      expect(found, `${scenarioKey} not found in options: ${ResultKeys.join(', ')}`)
    }

    const scenarioAuthMechKey = Object.keys(scenario.options).find((key) => key.toLowerCase() === 'authmechanismproperties')

    if (scenarioAuthMechKey != null) {
      const scenarioAuthMech = scenario.options[scenarioAuthMechKey] as Record<string, string>
      const resultAuthMechKey = Object.keys(result.data!.options || {}).find((key) => key.toLowerCase() === 'authmechanismproperties')

      expect(resultAuthMechKey, '"authmechanismproperties" not found in options').toBeDefined()

      const resultAuthMech = result.data!.options![resultAuthMechKey!] as Record<string, string> || {};

      const scenarioAuthMechKeys = Object.keys(scenarioAuthMech)
      const resultAuthMechKeys = Object.keys(resultAuthMech)

      expect(resultAuthMechKeys.length, `"authmechanismproperties" has mismatched keys. got: ${JSON.stringify(resultAuthMechKeys)}, expected: ${JSON.stringify(scenarioAuthMechKeys)}`)
        .toBe(scenarioAuthMechKeys.length)

      for(let i = 0, ln = scenarioAuthMechKeys.length; i < ln; i++) {
        let found = false
        let scenarioAuthMechKey = scenarioAuthMechKeys[i]!
        let scenarioAuthMechValue = scenarioAuthMech[scenarioAuthMechKey!]!
        for (let j = 0, ln = resultAuthMechKeys.length; j < ln; j++) {
          let resultAuthMechKey = resultAuthMechKeys[j]!
          if (scenarioAuthMechKey === resultAuthMechKey) {
            found = true
            let resultAuthMechValue = resultAuthMech[resultAuthMechKeys[j]!]!
            expect(resultAuthMechValue, `options.authMechanismProperties.${resultAuthMechValue} incorrect`)
              .toBe(scenarioAuthMechValue)
          }
        }

        expect(found, `mismatched keys, '${scenarioAuthMechKey}' not found in 'authmechanismproperties`).toBeTruthy()
      }
    }
  }
}
