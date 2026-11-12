'use strict';
const $ = id => document.getElementById(id);
const token = document.querySelector('meta[name="bridge-token"]').content;
let review = null;
function message(text, error = false) { $('message').textContent = text; $('message').classList.toggle('error', error); }
function step(number) { document.querySelectorAll('.steps li').forEach((el, index) => { if(index === number - 1) el.setAttribute('aria-current','step'); else el.removeAttribute('aria-current'); }); }
async function request(path, body, json = false) {
 const headers = {'X-Bridge-Token': token}; if(json) headers['Content-Type'] = 'application/json';
 const response = await fetch(path, {method:'POST', headers, body:json ? JSON.stringify(body) : body});
 const data = await response.json(); if(!response.ok) throw new Error(data.error || 'The bridge could not complete this request.'); return data;
}
function invalidate() { review=null; $('review-panel').hidden=true; $('result-panel').hidden=true; $('api-key').value=''; $('confirmed').checked=false; step(1); }
$('import-form').addEventListener('change', () => { invalidate(); const count=$('import-form').elements.files.files.length; $('selected').textContent=count ? `${count} export${count===1?'':'s'} selected.` : 'No files selected.'; message(''); });
$('import-form').addEventListener('submit', async event => {
 event.preventDefault(); invalidate(); const button=event.submitter; button.disabled=true; message('Reading your exports on this computer…');
 try {
  review = await request('/import', new FormData(event.target));
  $('counts').replaceChildren(); for(const text of [`${review.export_count} export${review.export_count===1?"":"s"}`,`${review.records} return${review.records===1?"":"s"}`,`${review.payer_batches} payer batch${review.payer_batches===1?"":"es"}`]) {const span=document.createElement('span');span.textContent=text;$('counts').append(span);}
  $('forms').replaceChildren(); for(const [form,count] of Object.entries(review.form_counts)) {const row=document.createElement('tr');for(const value of [form,count]){const cell=document.createElement('td');cell.textContent=value;row.append(cell);}$('forms').append(row);}
  $('totals').replaceChildren(); for(const [field,amount] of Object.entries(review.amount_field_sums)) {const dt=document.createElement('dt');dt.textContent=field;const dd=document.createElement('dd');dd.textContent=amount;$('totals').append(dt,dd);}
  $('review-panel').hidden=false;step(2);message('Imports reviewed locally. Form validation happens in TaxComposer when you prepare.');$('review-title').focus();
 } catch(error) {message(error.message,true);} finally {button.disabled=false;}
});
$('prepare-form').addEventListener('submit', async event => {
 event.preventDefault(); if(!review) return;
 const button=event.submitter; button.disabled=true; $('import-form').querySelectorAll('input,select,button').forEach(el=>el.disabled=true); message('Validating and preparing your returns in TaxComposer. Keep this window open…');
 const body={key:$('api-key').value,confirmed:$('confirmed').checked,source_sha256:review.source_sha256}; $('api-key').value='';
 try {
  const result=await request('/prepare',body,true);$('review-panel').hidden=true;$('result-panel').hidden=false;$('result-summary').textContent=`${result.prepared_batches} payer batches prepared. Nothing has been sent to the IRS.`;
  $('continue').href=result.filing_service;step(3);message('Drafts prepared. Continue in TaxComposer for review and sending.');$('result-title').focus();
 } catch(error) {message(error.message,true);} finally {body.key='';button.disabled=false;$('import-form').querySelectorAll('input,select,button').forEach(el=>el.disabled=false);}
});
$('clear').addEventListener('click', async () => {try{await request('/clear',{} ,true);$('import-form').reset();invalidate();$('selected').textContent='No files selected.';message('Ready for another import.');$('import-title').scrollIntoView();}catch(error){message(error.message,true);}});
