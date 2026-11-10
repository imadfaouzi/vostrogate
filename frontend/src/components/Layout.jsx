import React,{useEffect,useState} from 'react';
import {Link,NavLink,Outlet,useLocation,useNavigate} from 'react-router-dom';
import {api} from '../lib/api';
import {Icon,Button} from './Ui';

const menu = [
 {label:'Tableau de bord',icon:'dashboard',to:'/app/dashboard',perm:'dashboard.read'},
 {label:'Clients',icon:'users',to:'/app/clients',perm:'customers.read'},
 {label:'Fournisseurs',icon:'truck',to:'/app/fournisseurs',perm:'suppliers.read'},
 {label:'Articles',icon:'box',to:'/app/articles',perm:'products.read',items:[['Liste des articles','/app/articles'],['Nouvel article','/app/articles/new'],['Catégories, marques, unités','/app/catalogue']]},
 {label:'Achats',icon:'cart',to:'/app/achats',perm:'purchases.read',items:[['Commandes fournisseur','/app/ventes/PURCHASE_ORDER'],['Réceptions','/app/ventes/PURCHASE_RECEIPT'],['Factures fournisseur','/app/ventes/PURCHASE_INVOICE']]},
 {label:'Ventes',icon:'receipt',to:'/app/ventes/DELIVERY',perm:'sales.read',items:[['Devis','/app/ventes/QUOTE'],['Commandes clients','/app/ventes/ORDER'],['Bons de livraison','/app/ventes/DELIVERY'],['Factures','/app/ventes/INVOICE'],['Avoirs','/app/ventes/CREDIT'],['Retours des ventes','/app/ventes/RETURN']]},
 {label:'Inventaire',icon:'layers',to:'/app/inventaire',perm:'inventory.read'},
 {label:'Chèques / Effets',icon:'credit',to:'/app/cheques',perm:'cheques.read'},
 {label:'Paiements',icon:'wallet',to:'/app/paiements',perm:'payments.read'},
 {label:'Dépenses',icon:'wallet',to:'/app/depenses',perm:'expenses.read'},
 {label:'Rapports',icon:'chart',to:'/app/rapports',perm:'reports.read'},
 {label:'Point de vente',icon:'store',to:'/app/pos',perm:'pos.use'},
 {label:'Utilisateurs',icon:'user',to:'/app/utilisateurs',perm:'users.read'},
 {label:'Paramètres',icon:'settings',to:'/app/parametres',perm:'settings.read'},
];
const has=(user,permission)=>user?.owner||user?.permissions?.includes('*')||user?.permissions?.includes(permission);
export default function Layout({user,onLogout}) {
 const [company,setCompany]=useState(null);const [sidebar,setSidebar]=useState(false);const [opened,setOpened]=useState({});const loc=useLocation();const navigate=useNavigate();
 useEffect(()=>{api('company').then(v=>setCompany(v.company)).catch(()=>{})},[]);
 useEffect(()=>{setSidebar(false)},[loc.pathname]);
 return <div className="app-shell">
  {sidebar&&<div className="backdrop" onClick={()=>setSidebar(false)}/>}
  <aside className={'sidebar '+(sidebar?'visible':'')}>
   <Link to="/app/dashboard" className="brand"><img src="/brand/logo.svg" alt="Vostrogate"/><span><strong>Vostrogate</strong><small>ERP • Version 1.0.0</small></span></Link>
   <div className="company-chip"><span className="company-avatar"><img src="/api/company/logo" alt="" onError={e=>{e.currentTarget.style.display="none"}}/></span><span className="ellipsis"><small>Entreprise</small><strong>{company?.tradeName||company?.legalName||'Votre société'}</strong></span></div>
   <div className="side-section">MENU PRINCIPAL</div>
   <nav aria-label="Navigation principale">
    {menu.filter(item=>has(user,item.perm)).map(item=>{const expanded=opened[item.label]??(item.items?.some(([,url])=>loc.pathname===url));return <div key={item.label}>
      <div className="nav-row"><NavLink to={item.to} className={({isActive})=>'navitem '+(isActive||(!item.items&&loc.pathname.startsWith(item.to))?'active':'')}><Icon name={item.icon}/><span>{item.label}</span></NavLink>{item.items&&<button className="expand-btn" aria-label={'Développer '+item.label} aria-expanded={expanded} onClick={()=>setOpened(p=>({...p,[item.label]:!expanded}))}><Icon name="chevron" size={15}/></button>}</div>
      {item.items&&expanded&&<div className="subnav">{item.items.map(([label,to])=><NavLink to={to} key={to} className={({isActive})=>isActive?'active':''}>{label}</NavLink>)}</div>}
    </div>})}
   </nav>
   <div className="side-bottom"><div className="brand-status"><span className="status-dot"/> Base MySQL connectée</div><small>Vostrogate 1.0.0 • Une entreprise</small></div>
  </aside>
  <div className="main-area"><header className="topbar"><button className="mobile-menu icon-button" aria-label="Menu" onClick={()=>setSidebar(!sidebar)}><Icon name="menu"/></button><span className="breadcrumb">{company?.tradeName||company?.legalName||'Entreprise'}</span><span className="top-spacer"/><Link className="top-shortcut" to="/app/ventes/DELIVERY">Ventes</Link>{has(user,'pos.use')&&<Link className="top-shortcut pos-shortcut" to="/app/pos"><Icon name="store" size={15}/> Point de vente</Link>}<span className="user-pill"><span className="avatar">{user?.name?.[0]||'A'}</span><span><strong>{user?.name}</strong><small>{user?.role}</small></span></span><Button title="Se déconnecter" icon="log-out" onClick={onLogout}/></header><main className="main-content"><Outlet context={{user,company,setCompany}}/></main></div>
 </div>
}
