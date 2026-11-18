import 'fake-indexeddb/auto'
import { createStore, set } from 'idb-keyval'
import { describe, expect, it } from 'vitest'

import { IndexedDbDraftStore } from './indexedDbDraftStore'

let databaseNumber = 0

function freshStore() {
  databaseNumber += 1

  return createStore(`arqetra-test-${databaseNumber}`, 'drafts')
}

const draft = {
  name: 'Поле біля ставка',
  points: [
    { latitude: 50.45021, longitude: 30.52011 },
    { latitude: 50.45048, longitude: 30.52342 },
  ],
  saveAfterSignIn: true,
}

describe('IndexedDbDraftStore', () => {
  it('keeps the draft between page loads', async () => {
    const store = freshStore()

    await new IndexedDbDraftStore(store).save(draft)

    expect(await new IndexedDbDraftStore(store).load()).toEqual(draft)
  })

  it('has no draft at first and after clearing', async () => {
    const drafts = new IndexedDbDraftStore(freshStore())

    expect(await drafts.load()).toBeNull()
    await drafts.save(draft)
    await drafts.clear()
    expect(await drafts.load()).toBeNull()
  })

  it('ignores a damaged draft', async () => {
    const store = freshStore()
    await set('calculator', { name: 'x', points: [{ latitude: 'a' }] }, store)

    expect(await new IndexedDbDraftStore(store).load()).toBeNull()
  })
})
