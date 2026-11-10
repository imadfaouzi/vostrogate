import React from 'react';
import {Link} from 'react-router-dom';
import {Card,Icon,Loading,Page,Alert} from '../components/Ui';
import {useLoad} from '../hooks/useLoad';
import {money} from '../lib/api';
const quick=[['Nouveau devis','/app/documents/new?type=QUOTE','file'],['Bon de livraison','/app/documents/new?type=DELIVERY','receipt'],['Créer un article','/app/articles/new','box'],['Ajuster le stock','/app/inventaire','layers'],['Nouveau client','/app/clients/new','users'],['Rapports','/app/rapports','chart']];
export default function Dashboard(){
 const {data,loading,error}=useLoad('dashboard',[]);
 const kpis=[['Chiffre d’affaires',data?.salesCents,'chart'],['CA impayé',data?.unpaidCents,'wallet'],['Total achats',data?.purchasesCents,'cart'],['Dépenses',data?.expensesCents,'credit']];
 const trend=data?.salesTrend||[];const max=Math.max(1,...trend.map(x=>x.totalCents));
 return <Page title="Tableau de bord" description="Vue d'ensemble de votre activité commerciale" actions={<Link className="btn primary" to="/app/documents/new?type=DELIVERY"><Icon name="plus"/> Nouvelle vente</Link>}><Alert message={error}/>{loading?<Loading/>:<>
 <div className="kpi-grid">{kpis.map(([label,value,icon])=><div className="kpi" key={label}><div className="kpi-label">{label}<Icon name={icon} size={22}/></div><strong>{money(value||0)}</strong><small>Données en temps réel — MySQL</small></div>)}</div>
 <div className="grid grid-2"><Card title="Ventes des 30 derniers jours">{trend.length?<div className="chart" role="img" aria-label="Barres de ventes par jour">{trend.map(item=><div key={item.date} className="chart-bar" style={{height:Math.max(3,Math.round(100*item.totalCents/max))+'%'}} title={`${item.date} : ${money(item.totalCents)}`}><span>{item.date.slice(8)}</span></div>)}</div>:<p className="muted">Aucune vente validée sur cette période.</p>}</Card>
 <Card title="Raccourcis"><div className="quick-grid">{quick.map(([label,to,icon])=><Link key={label} to={to} className="quick-link"><Icon name={icon}/>{label}</Link>)}</div><div className="mini-stat"><span className="muted">Clients</span><strong>{data?.customers}</strong></div><div className="mini-stat"><span className="muted">Articles</span><strong>{data?.products}</strong></div><div className="mini-stat"><span className="muted">Chèques proches d'échéance</span><Link to="/app/cheques"><strong>{data?.chequeAlerts}</strong></Link></div></Card></div></>}
 </Page>;
}
