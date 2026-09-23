const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const {test} = require('node:test');
const {translate} = require('../i18n.js');

test('Russian and Kazakh translations preserve dynamic numbers and unknown content', () => {
  assert.equal(translate('My Career', 'ru'), 'Моя карьера');
  assert.equal(translate('My Career', 'kk'), 'Менің мансабым');
  assert.equal(translate('12 study hours', 'ru'), '12 часов обучения');
  assert.equal(translate('Step 2 · Start here', 'kk'), '2-қадам · Осы жерден бастаңыз');
  assert.equal(translate('＋ Create event', 'ru'), '＋ Создать мероприятие');
  assert.equal(translate('SQL Foundations', 'kk'), 'SQL Foundations');
  assert.equal(translate('My Career', 'en'), 'My Career');
});

test('switching languages preserves option values and handles newly rendered text', () => {
  let change, update, mounted;
  const saved = new Map([['careerquest.language', 'ru']]);
  const parent = {closest: () => null};
  const title = {nodeValue: 'My Career', parentElement: parent};
  const optionText = {nodeValue: 'Junior', parentElement: parent};
  const nodes = [title, optionText];
  const attrs = {};
  const option = {
    get textContent() {return optionText.nodeValue;},
    get value() {return attrs.value ?? optionText.nodeValue;},
    setAttribute(key, value) {attrs[key] = value;},
  };
  const select = {value: '', addEventListener(type, fn) {change = fn;}};
  const host = {matches: () => false, append(value) {mounted = value;}};
  const document = {
    body: {}, documentElement: {}, title: 'My Career', addEventListener() {},
    createElement: () => ({setAttribute() {}, querySelector: () => select}),
    querySelector: () => host,
    querySelectorAll: selector => selector.startsWith('option') && !attrs.value ? [option] : [],
    createTreeWalker: () => {
      let index = -1;
      return {nextNode() {return ++index < nodes.length;}, get currentNode() {return nodes[index];}};
    },
  };
  const context = vm.createContext({document, NodeFilter: {SHOW_TEXT: 4},
    localStorage: {getItem: key => saved.get(key), setItem: (key, value) => saved.set(key, value)},
    MutationObserver: class {constructor(fn) {update = fn;} disconnect() {} observe() {}},
  });
  vm.runInContext(fs.readFileSync(require.resolve('../i18n.js'), 'utf8'), context);
  vm.runInContext('CareerQuestI18n.init()', context);
  assert.ok(mounted);
  assert.equal(title.nodeValue, 'Моя карьера');
  assert.equal(option.value, 'Junior');
  select.value = 'kk'; change();
  assert.equal(title.nodeValue, 'Менің мансабым');
  assert.equal(option.value, 'Junior');
  assert.equal(document.documentElement.lang, 'kk');
  assert.equal(saved.get('careerquest.language'), 'kk');
  title.nodeValue = 'Save goal'; update();
  assert.equal(title.nodeValue, 'Мақсатты сақтау');
  select.value = 'en'; change();
  assert.equal(title.nodeValue, 'Save goal');
  assert.equal(optionText.nodeValue, 'Junior');
  assert.equal(option.value, 'Junior');
});
