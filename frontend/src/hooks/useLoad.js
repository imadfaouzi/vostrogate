import { useEffect, useState } from 'react';
import { api } from '../lib/api';
export function useLoad(path, deps=[]) {
  const [data,setData]=useState(null);const [loading,setLoading]=useState(true);const [error,setError]=useState('');
  useEffect(()=>{let ok=true;const c=new AbortController();setLoading(true);setError('');
    api(path,{signal:c.signal}).then(x=>{if(ok)setData(x)}).catch(e=>{if(ok&&e.name!=='AbortError')setError(e.message)}).finally(()=>{if(ok)setLoading(false)});
    return ()=>{ok=false;c.abort()};
  // eslint-disable-next-line react-hooks/exhaustive-deps
  },deps);
  return {data,loading,error,setData};
}
export function useDebounced(value,delay=200) {
  const [v,setV]=useState(value);useEffect(()=>{let t=setTimeout(()=>setV(value),delay);return()=>clearTimeout(t)},[value,delay]);return v;
}
