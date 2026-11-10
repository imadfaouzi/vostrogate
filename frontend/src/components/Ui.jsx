import React, { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, getError, money, qs } from '../lib/api';
export const Icon=({name,size=19,...props})=><svg className="vg-icon" width={size} height={size} fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" {...props}><use href={'/brand/icons.svg#i-'+name}/></svg>;
export const Button=({children,icon,variant='default',...props})=><button className={'btn '+(variant==='default'?'':variant)} {...props}>{icon&&<Icon name={icon} size={17}/>} {children}</button>;
export const Page=({title,description,actions,children})=><><header className="page-header"><div><p className="eyebrow">Vostrogate / Gestion</p><h1>{title}</h1>{description&&<p className="muted">{description}</p>}</div><div className="actions">{actions}</div></header>{children}</>;
export const Card=({title,actions,children,className=''})=><section className={'card '+className}>{title&&<div className="card-title"><h2>{title}</h2><div className="actions">{actions}</div></div>}{children}</section>;
export const Field=({label,children,hint,required})=><label className="field"><span>{label}{required&&<strong className="required"> *</strong>}</span>{children}{hint&&<small className="muted">{hint}</small>}</label>;
export const Input=(props)=><input className="input" {...props}/>;
export const Select=({options=[],...props})=><select className="input" {...props}>{options.map(o=><option key={typeof o==='object'?o.value:o} value={typeof o==='object'?o.value:o}>{typeof o==='object'?o.label:o}</option>)}</select>;
export const Textarea=(props)=><textarea className="input" rows={3} {...props}/>;
export const Alert=({message,kind='error'})=>message?<div role="alert" className={'alert '+kind}>{message}</div>:null;
export const Badge=({status})=><span className={'tag '+({FINAL:'success',DRAFT:'warn',CANCELLED:'danger',CLEARED:'success',PENDING:'warn',REJECTED:'danger'}[status]||'neutral')}>{({FINAL:'Validé',DRAFT:'Brouillon',CANCELLED:'Annulé',CLEARED:'Encaissé',PENDING:'En attente',REJECTED:'Rejeté'}[status]||status)}</span>;
export const Loading=()=> <div className="loading"><span className="spinner"/> Chargement...</div>;
export const Empty=({message='Aucun résultat trouvé'})=><div className="empty"><Icon name="box" size={36}/><p>{message}</p></div>;
export const Toolbar=({q,setQ,placeholder='Rechercher...',other})=><div className="toolbar"><div className="searchfield"><Icon name="search" size={18}/><input value={q} onChange={e=>setQ(e.target.value)} placeholder={placeholder}/></div>{other}</div>;
export const Pager=({offset,setOffset,total,limit=50})=><div className="pager"><span className="muted">{total} élément(s) — page {Math.floor(offset/limit)+1} / {Math.max(1,Math.ceil(total/limit))}</span><div className="actions"><Button disabled={offset<=0} onClick={()=>setOffset(Math.max(0,offset-limit))} icon="left">Précédent</Button><Button disabled={offset+limit>=total} onClick={()=>setOffset(offset+limit)}>Suivant <Icon name="right" size={15}/></Button></div></div>;
export const FormActions=({busy,submit='Enregistrer',cancel})=><div className="form-actions">{cancel&&<Button type="button" onClick={cancel}>Annuler</Button>}<Button variant="primary" icon="save" type="submit" disabled={busy}>{busy?'Enregistrement...':submit}</Button></div>;
export function ProductAutocomplete({onSelect,placeholder='Nom, référence ou code-barres...',autoFocus=false}){
  const [text,setText]=useState('');const [rows,setRows]=useState([]);const [open,setOpen]=useState(false);const [active,setActive]=useState(0);const [busy,setBusy]=useState(false);const ref=useRef(null);const timer=useRef(0);
  useEffect(()=>{const close=(e)=>{if(!ref.current?.contains(e.target))setOpen(false)};document.addEventListener('mousedown',close);return()=>document.removeEventListener('mousedown',close)},[]);
  useEffect(()=>{const val=text.trim();const controller=new AbortController();clearTimeout(timer.current);
    if(!val){setRows([]);setOpen(false);return()=>controller.abort()}
    timer.current=setTimeout(()=>{setBusy(true);api('products/search?'+qs({q:val}),{signal:controller.signal}).then(res=>{setRows(res.items||[]);setOpen(true);setActive(0)}).catch(e=>{if(e.name!=='AbortError')setRows([])}).finally(()=>setBusy(false))},190);
    return()=>{clearTimeout(timer.current);controller.abort()};
  },[text]);
  const choose=(item)=>{onSelect(item);setText('');setOpen(false);setRows([])};
  const keyDown=(e)=>{if(e.key==='ArrowDown'){e.preventDefault();setActive(v=>Math.min(v+1,rows.length-1));setOpen(true)}else if(e.key==='ArrowUp'){e.preventDefault();setActive(v=>Math.max(v-1,0))}else if(e.key==='Enter'&&open&&rows.length){e.preventDefault();choose(rows[active]||rows[0])}else if(e.key==='Escape'){setOpen(false)}};
  return <div className="autocomplete" ref={ref}>
    <div className="searchfield product-search"><Icon name="search"/><input role="combobox" aria-expanded={open} aria-controls="vg-product-results" aria-autocomplete="list" aria-activedescendant={open?`vg-product-${active}`:undefined} value={text} autoFocus={autoFocus} onChange={e=>setText(e.target.value)} onFocus={()=>{if(rows.length)setOpen(true)}} onKeyDown={keyDown} placeholder={placeholder} autoComplete="off"/>{busy&&<span className="spinner tiny"/>}</div>
    {open&&<div className="suggestions" id="vg-product-results" role="listbox">{rows.length?rows.map((item,i)=><button key={item.id} id={`vg-product-${i}`} className={'suggestion '+(active===i?'selected':'')} role="option" aria-selected={active===i} type="button" onMouseDown={e=>e.preventDefault()} onClick={()=>choose(item)}><span><strong>{item.name}</strong><small>{item.sku}{item.barcode&&' · '+item.barcode}</small></span><span className="suggestion-right"><b>{money(item.salePriceCents)}</b><small>Stock: {(item.stockMilli/1000).toLocaleString('fr-FR')}</small></span></button>):<div className="noresults">Aucun article. Vérifiez le nom ou le code-barres.</div>}</div>}
  </div>;
}
export function CsvButton({rows,filename='export.csv'}){
 const exportRows=()=>{if(!rows?.length)return;const keys=Object.keys(rows[0]);const cell=(v)=>'"'+String(v??'').replaceAll('"','""')+'"';const csv='\ufeff'+[keys.join(';'),...rows.map(o=>keys.map(k=>cell(o[k])).join(';'))].join('\r\n');const blob=new Blob([csv],{type:'text/csv;charset=utf-8'});const url=URL.createObjectURL(blob);const link=document.createElement('a');link.href=url;link.download=filename;link.click();URL.revokeObjectURL(url)};
 return <Button icon="download" type="button" onClick={exportRows} disabled={!rows?.length}>CSV</Button>;
}
export const ErrorBox=({error})=><Alert message={typeof error==='string'?error:getError(error)}/>;
export const ActionLink=({to,children,icon})=><Link to={to} className="btn primary">{icon&&<Icon name={icon}/>} {children}</Link>;
