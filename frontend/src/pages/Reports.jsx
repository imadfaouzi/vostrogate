import React,{useState} from 'react';
import {Link} from 'react-router-dom';
import {api,formatDate,money,qs} from '../lib/api';
import {useLoad} from '../hooks/useLoad';
import {Alert,Button,Card,CsvButton,Empty,Field,Input,Loading,Page,Select} from '../components/Ui';
const kinds=[{value:'sales',label:'Rapport des ventes'},{value:'purchases',label:'Rapport des achats'},{value:'receivables',label:'Soldes clients'},{value:'stock',label:'État des stocks'},{value:'expenses',label:'Dépenses'}];
export default function Reports({audit=false}){
 const [kind,setKind]=useState('sales');const [from,setFrom]=useState(new Date(Date.now()-30*86400000).toISOString().slice(0,10));const [to,setTo]=useState(new Date().toISOString().slice(0,10));const [run,setRun]=useState(0);
 const url=audit?'audit':'reports?'+qs({kind,from,to});const {data,loading,error}=useLoad(url,[audit,run]);const rows=data?.items||[];
 const cols=audit?['date','user','action','entity','entityId']:kind==='receivables'?['code','name','balanceCents']:kind==='stock'?['sku','name','warehouse','quantityMilli']:kind==='expenses'?['date','title','category','amountCents']:['date','number','partyName','totalCents','paidCents'];
 const labels={date:'Date',user:'Utilisateur',action:'Opération',entity:'Entité',entityId:'ID',code:'Code',name:'Nom',balanceCents:'Reste dû',sku:'Référence',warehouse:'Dépôt',quantityMilli:'Stock',title:'Libellé',category:'Catégorie',amountCents:'Montant',number:'N°',partyName:'Partenaire',totalCents:'Total TTC',paidCents:'Réglé'};
 const fmt=(k,v)=>k.endsWith('Cents')?money(v):k==='quantityMilli'?(v/1000).toLocaleString('fr-FR'):k==='date'?formatDate(v):String(v??'—');
 return <Page title={audit?'Journal d’audit':'Rapports'} description={audit?'Historique des activités sensibles':'Analyses commerciales, finances, stock et export CSV'} actions={audit?null:<CsvButton rows={rows} filename={'rapport-'+kind+'.csv'}/> }><div className="grid" style={{gridTemplateColumns:audit?'1fr':'250px minmax(0,1fr)'}}>
 {!audit&&<Card title="Type de rapport"><div className="stack">{kinds.map(k=><button key={k.value} className={'btn '+(kind===k.value?'primary':'')} style={{justifyContent:'flex-start'}} onClick={()=>{setKind(k.value);setRun(x=>x+1)}}>{k.label}</button>)}</div></Card>}
 <Card title={audit?'Historique des accès et modifications':kinds.find(k=>k.value===kind)?.label}>
 {!audit&&<div className="toolbar no-print"><Field label="Du"><Input type="date" value={from} onChange={e=>setFrom(e.target.value)}/></Field><Field label="Au"><Input type="date" value={to} onChange={e=>setTo(e.target.value)}/></Field><Button variant="primary" onClick={()=>setRun(n=>n+1)}>Générer le rapport</Button><Button icon="print" onClick={()=>window.print()}>PDF / Imprimer</Button></div>}
 <Alert message={error}/>{loading?<Loading/>:rows.length?<><div className="table-wrap"><table className="table"><thead><tr>{cols.map(c=><th key={c}>{labels[c]||c}</th>)}</tr></thead><tbody>{rows.map((r,i)=><tr key={r.id||i}>{cols.map(c=><td key={c}>{fmt(c,r[c])}</td>)}</tr>)}</tbody></table></div>{['sales','purchases'].includes(kind)&&!audit&&<div className="totals" style={{marginTop:15}}><div><span>Total période</span><strong>{money(data?.totalCents)}</strong></div><div><span>Réglé</span><strong>{money(data?.paidCents)}</strong></div></div>}</>:<Empty message="Aucune donnée pour ce rapport."/>}</Card></div></Page>;
}
