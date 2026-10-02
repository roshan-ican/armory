//#region node_modules/svelte/src/internal/shared/utils.js
var e = Array.isArray, t = Array.prototype.indexOf, n = Array.prototype.includes, r = Array.from, i = Object.defineProperty, a = Object.getOwnPropertyDescriptor, o = Object.getOwnPropertyDescriptors, s = Object.prototype, c = Array.prototype, l = Object.getPrototypeOf, u = Object.isExtensible, d = () => {};
function f(e) {
	return e();
}
function p(e) {
	for (var t = 0; t < e.length; t++) e[t]();
}
function m() {
	var e, t;
	return {
		promise: new Promise((n, r) => {
			e = n, t = r;
		}),
		resolve: e,
		reject: t
	};
}
var h = 1024, g = 2048, _ = 4096, v = 8192, y = 16384, b = 32768, x = 1 << 25, S = 65536, ee = 1 << 18, C = 1 << 19, te = 1 << 20, ne = 1 << 25, re = 1 << 21, ie = 1 << 22, w = 1 << 23, ae = Symbol("$state"), oe = Symbol("component"), se = Symbol(""), ce = Symbol("attributes"), le = Symbol("class"), ue = Symbol("style"), de = Symbol("text"), fe = Symbol("form reset"), pe = new class extends Error {
	name = "StaleReactionError";
	message = "The reaction that called `getAbortSignal()` was re-run or destroyed";
}(), me = !!globalThis.document?.contentType && /* @__PURE__ */ globalThis.document.contentType.includes("xml"), he = {}, T = Symbol("uninitialized"), ge = "http://www.w3.org/1999/xhtml";
function _e() {
	console.warn("https://svelte.dev/e/derived_inert");
}
function ve(e) {
	console.warn("https://svelte.dev/e/hydration_mismatch");
}
function ye() {
	console.warn("https://svelte.dev/e/svelte_boundary_reset_noop");
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/hydration.js
var E = !1;
function be(e) {
	E = e;
}
var D;
function O(e) {
	if (e === null) throw ve(), he;
	return D = e;
}
function xe() {
	return O(/* @__PURE__ */ tn(D));
}
function k(e) {
	if (E) {
		if (/* @__PURE__ */ tn(D) !== null) throw ve(), he;
		D = e;
	}
}
function Se(e = 1) {
	if (E) {
		for (var t = e, n = D; t--;) n = /* @__PURE__ */ tn(n);
		D = n;
	}
}
function Ce(e = !0) {
	for (var t = 0, n = D;;) {
		if (n.nodeType === 8) {
			var r = n.data;
			if (r === "]") {
				if (t === 0) return n;
				--t;
			} else (r === "[" || r === "[!" || r[0] === "[" && !isNaN(Number(r.slice(1)))) && (t += 1);
		}
		var i = /* @__PURE__ */ tn(n);
		e && n.remove(), n = i;
	}
}
function we(e) {
	if (!e || e.nodeType !== 8) throw ve(), he;
	return e.data;
}
//#endregion
//#region node_modules/svelte/src/internal/client/reactivity/equality.js
function Te(e) {
	return e === this.v;
}
function Ee(e, t) {
	return e == e ? e !== t || typeof e == "object" && !!e || typeof e == "function" : t == t;
}
function De(e) {
	return !Ee(e, this.v);
}
function Oe(e) {
	throw Error("https://svelte.dev/e/lifecycle_outside_component");
}
//#endregion
//#region node_modules/svelte/src/internal/client/errors.js
function ke() {
	throw Error("https://svelte.dev/e/async_derived_orphan");
}
function Ae(e, t, n) {
	throw Error("https://svelte.dev/e/each_key_duplicate");
}
function je(e) {
	throw Error("https://svelte.dev/e/effect_in_teardown");
}
function Me() {
	throw Error("https://svelte.dev/e/effect_in_unowned_derived");
}
function Ne(e) {
	throw Error("https://svelte.dev/e/effect_orphan");
}
function Pe() {
	throw Error("https://svelte.dev/e/effect_update_depth_exceeded");
}
function Fe() {
	throw Error("https://svelte.dev/e/state_descriptors_fixed");
}
function Ie() {
	throw Error("https://svelte.dev/e/state_prototype_fixed");
}
function Le() {
	throw Error("https://svelte.dev/e/state_unsafe_mutation");
}
function Re() {
	throw Error("https://svelte.dev/e/svelte_boundary_reset_onerror");
}
//#endregion
//#region node_modules/svelte/src/internal/flags/index.js
var ze = !1;
function Be() {
	ze = !0;
}
//#endregion
//#region node_modules/svelte/src/internal/client/context.js
var A = null;
function Ve(e) {
	A = e;
}
function He(e, t = !1, n) {
	A = {
		p: A,
		i: !1,
		c: null,
		e: null,
		s: e,
		x: null,
		r: G,
		l: ze && !t ? {
			s: null,
			u: null,
			$: []
		} : null
	};
}
function Ue(e) {
	var t = A, n = t.e;
	if (n !== null) {
		t.e = null;
		for (var r of n) hn(r);
	}
	return e !== void 0 && (t.x = e), t.i = !0, A = t.p, We(e);
}
function We(e = {}) {
	return i(e, oe, { value: !0 }), e;
}
function Ge() {
	return !ze || A !== null && A.l === null;
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/task.js
var Ke = [];
function qe() {
	var e = Ke;
	Ke = [], p(e);
}
function Je(e) {
	if (Ke.length === 0 && !xt) {
		var t = Ke;
		queueMicrotask(() => {
			t === Ke && qe();
		});
	}
	Ke.push(e);
}
function Ye() {
	for (; Ke.length > 0;) qe();
}
//#endregion
//#region node_modules/svelte/src/internal/client/reactivity/status.js
var Xe = ~(g | _ | h);
function j(e, t) {
	e.f = e.f & Xe | t;
}
function Ze(e) {
	e.f & 512 || e.deps === null ? j(e, h) : j(e, _);
}
//#endregion
//#region node_modules/svelte/src/internal/client/reactivity/utils.js
function Qe(e, t, n) {
	e.f & 2048 ? t.add(e) : e.f & 4096 && n.add(e), j(e, h);
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/elements/misc.js
var $e = !1;
function et() {
	$e || ($e = !0, document.addEventListener("reset", (e) => {
		Promise.resolve().then(() => {
			if (!e.defaultPrevented) for (let t of e.target.elements) t[fe]?.();
		});
	}, { capture: !0 }));
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/elements/bindings/shared.js
function tt(e) {
	var t = U, n = G;
	W(null), Ln(null);
	try {
		return e();
	} finally {
		W(t), Ln(n);
	}
}
function nt(e, t, n, r = n) {
	e.addEventListener(t, () => tt(n));
	let i = e[fe];
	e[fe] = i ? () => {
		i(), r(!0);
	} : () => r(!0), et();
}
//#endregion
//#region node_modules/svelte/src/internal/client/reactivity/async.js
function rt(e, t, n, r) {
	let i = Ge() ? st : dt;
	var a = e.filter((e) => !e.settled), o = t.map(i);
	if (n.length === 0 && a.length === 0) {
		r(o);
		return;
	}
	var s = G, c = it(), l = a.length === 1 ? a[0].promise : a.length > 1 ? Promise.all(a.map((e) => e.promise)) : null;
	function u(e) {
		if (!(s.f & 16384)) {
			c();
			try {
				r([...o, ...e]);
			} catch (e) {
				cn(e, s);
			}
			at();
		}
	}
	var d = ot();
	if (n.length === 0) {
		l.then(() => u([])).finally(d);
		return;
	}
	function f() {
		Promise.all(n.map((e) => /* @__PURE__ */ lt(e))).then(u).catch((e) => cn(e, s)).finally(d);
	}
	l ? l.then(() => {
		c(), f(), at();
	}) : f();
}
function it() {
	var e = G, t = U, n = A, r = M;
	return function(i = !0) {
		Ln(e), W(t), Ve(n), i && !(e.f & 16384) && (r?.activate(), r?.apply());
	};
}
function at(e = !0) {
	Ln(null), W(null), Ve(null), e && M?.deactivate();
}
function ot() {
	var e = G, t = e.b, n = M, r = !!t?.is_rendered();
	return t?.update_pending_count(1, n), n.increment(r, e), () => {
		t?.update_pending_count(-1, n), n.decrement(r, e);
	};
}
/*#__NO_SIDE_EFFECTS__*/
function st(e) {
	var t = 2 | g;
	return G !== null && (G.f |= C), {
		ctx: A,
		deps: null,
		effects: null,
		equals: Te,
		f: t,
		fn: e,
		reactions: null,
		rv: 0,
		v: T,
		wv: 0,
		parent: G,
		ac: null
	};
}
var ct = Symbol("obsolete");
/*#__NO_SIDE_EFFECTS__*/
function lt(e, t, n) {
	let r = G;
	r === null && ke();
	var i = void 0, a = Rt(T), o = !U, s = /* @__PURE__ */ new Set();
	return yn(() => {
		var t = G, n = m();
		i = n.promise;
		try {
			Promise.resolve(e()).then(n.resolve, (e) => {
				e !== pe && n.reject(e);
			}).finally(at);
		} catch (e) {
			n.reject(e), at();
		}
		var c = M;
		if (o) {
			if (t.f & 32768) var l = ot();
			if (r.b?.is_rendered()) c.async_deriveds.get(t)?.reject(ct);
			else for (let e of s.values()) e.reject(ct);
			s.add(n), c.async_deriveds.set(t, n);
		}
		let u = (e, t = void 0) => {
			l?.(), s.delete(n), t !== ct && (c.activate(), t ? (a.f |= w, Ut(a, t)) : (a.f & 8388608 && (a.f ^= w), Ut(a, e)), c.deactivate());
		};
		n.promise.then(u, (e) => u(null, e || "unknown"));
	}), pn(() => {
		for (let e of s) e.reject(ct);
	}), new Promise((e) => {
		function t(n) {
			function r() {
				n === i ? e(a) : t(i);
			}
			n.then(r, r);
		}
		t(i);
	});
}
/*#__NO_SIDE_EFFECTS__*/
function ut(e) {
	let t = /* @__PURE__ */ st(e);
	return zn(t), t;
}
/*#__NO_SIDE_EFFECTS__*/
function dt(e) {
	let t = /* @__PURE__ */ st(e);
	return t.equals = De, t;
}
function ft(e) {
	var t = e.effects;
	if (t !== null) {
		e.effects = null;
		for (var n = 0; n < t.length; n += 1) H(t[n]);
	}
}
function pt(e) {
	var t, n = G, r = e.parent;
	if (!Pn && r !== null && e.v !== T && r.f & 24576) return _e(), e.v;
	Ln(r);
	try {
		ft(e), t = Jn(e);
	} finally {
		Ln(n);
	}
	return t;
}
function mt(e) {
	var t = pt(e);
	if (!e.equals(t) && (e.wv = Gn(), (!M?.is_fork || e.deps === null) && (M === null ? e.v = t : (M.capture(e, t, !0), vt?.capture(e, t, !0)), e.deps === null))) {
		j(e, h);
		return;
	}
	Pn || (yt === null ? Ze(e) : (fn() || M?.is_fork) && yt.set(e, t));
}
function ht(e) {
	if (e.effects !== null) for (let t of e.effects) (t.teardown || t.ac) && (t.teardown?.(), t.ac !== null && tt(() => {
		t.ac.abort(pe), t.ac = null;
	}), t.fn !== null && (t.teardown = d), Zn(t, 0), Cn(t));
}
function gt(e) {
	if (e.effects !== null) for (let t of e.effects) t.teardown && t.fn !== null && Qn(t);
}
//#endregion
//#region node_modules/svelte/src/internal/client/reactivity/batch.js
var _t = null, M = null, vt = null, yt = null, bt = null, xt = !1, St = !1, Ct = null, wt = null, Tt = 0, Et = 1, Dt = class e {
	id = Et++;
	#e = !1;
	linked = !0;
	#t = null;
	#n = null;
	async_deriveds = /* @__PURE__ */ new Map();
	current = /* @__PURE__ */ new Map();
	previous = /* @__PURE__ */ new Map();
	#r = /* @__PURE__ */ new Set();
	#i = /* @__PURE__ */ new Set();
	#a = 0;
	#o = /* @__PURE__ */ new Map();
	#s = null;
	#c = [];
	#l = [];
	#u = /* @__PURE__ */ new Set();
	#d = /* @__PURE__ */ new Set();
	#f = /* @__PURE__ */ new Map();
	#p = /* @__PURE__ */ new Set();
	is_fork = !1;
	#m = !1;
	constructor() {
		_t === null ? _t = this : (_t.#n = this, this.#t = _t), _t = this;
	}
	#h() {
		if (this.is_fork) return !0;
		for (let n of this.#o.keys()) {
			for (var e = n, t = !1; e.parent !== null;) {
				if (this.#f.has(e)) {
					t = !0;
					break;
				}
				e = e.parent;
			}
			if (!t) return !0;
		}
		return !1;
	}
	skip_effect(e) {
		this.#f.has(e) || this.#f.set(e, {
			d: [],
			m: []
		}), this.#p.delete(e);
	}
	unskip_effect(e, t = (e) => this.schedule(e)) {
		var n = this.#f.get(e);
		if (n) {
			this.#f.delete(e);
			for (var r of n.d) j(r, g), t(r);
			for (r of n.m) j(r, _), t(r);
		}
		this.#p.add(e);
	}
	#g() {
		var e = [];
		for (let i of this.#c) if (!(i.f & 16384 || !(i.f & 6144))) {
			for (var t = i, n = !1; t.parent !== null;) {
				t = t.parent;
				var r = t.f;
				if (r & 96) {
					if (!(r & 1024)) {
						n = !0;
						break;
					}
					t.f ^= h;
				}
			}
			n || e.push(t);
		}
		return this.#c = [], e;
	}
	#_() {
		this.#e = !0;
		for (let e of this.#u) this.#d.delete(e), j(e, g), this.schedule(e);
		for (let e of this.#d) j(e, _), this.schedule(e);
		this.apply();
		for (var t = Ct = [], n = [], r = wt = []; this.#c.length > 0;) {
			Tt++ > 1e3 && (this.#S(), kt());
			for (let e of this.#g()) try {
				this.#v(e, t, n);
			} catch (t) {
				throw Pt(e), this.#h() || this.discard(), t;
			}
		}
		if (M = null, r.length > 0) {
			var i = e.ensure();
			for (let e of r) i.schedule(e);
		}
		if (Ct = null, wt = null, this.#h()) {
			this.#x(n), this.#x(t);
			for (let [e, t] of this.#f) Nt(e, t);
			r.length > 0 && M.#_();
			return;
		}
		let a = this.#y();
		if (a) {
			this.#x(n), this.#x(t), a.#b(this);
			return;
		}
		this.#u.clear(), this.#d.clear();
		for (let e of this.#r) e(this);
		this.#r.clear(), vt = this, jt(n), jt(t), vt = null, this.#s?.resolve();
		var o = M;
		if (this.#a === 0 && (this.#c.length === 0 || o !== null) && this.#S(), this.#c.length > 0) {
			if (o !== null) {
				for (let e of this.#c) o.#c.push(e);
				this.#c = [];
			} else o = this;
		}
		o !== null && (It.clear(), o.#_());
	}
	#v(e, t, n) {
		e.f ^= h;
		for (var r = e.first; r !== null;) {
			var i = r.f, a = !!(i & 96);
			if (!(a && i & 1024 || i & 8192 || this.#f.has(r)) && r.fn !== null) {
				a ? r.f ^= h : i & 4 ? t.push(r) : Kn(r) && (i & 16 && this.#d.add(r), Qn(r));
				var o = r.first;
				if (o !== null) {
					r = o;
					continue;
				}
			}
			for (; r !== null;) {
				var s = r.next;
				if (s !== null) {
					r = s;
					break;
				}
				r = r.parent;
			}
		}
	}
	#y() {
		for (var e = this.#t; e !== null;) {
			if (!e.is_fork) {
				for (let [t, [, n]] of this.current) if (e.current.has(t) && !n) return e;
			}
			e = e.#t;
		}
		return null;
	}
	#b(e) {
		for (let [t, n] of e.current) !this.previous.has(t) && e.previous.has(t) && this.previous.set(t, e.previous.get(t)), this.current.set(t, n);
		for (let [t, n] of e.async_deriveds) {
			let e = this.async_deriveds.get(t);
			e && n.promise.then(e.resolve).catch(e.reject);
		}
		e.async_deriveds.clear(), this.transfer_effects(e.#u, e.#d);
		let t = (e) => {
			var n = e.reactions;
			if (n !== null && !(e.f & 2 && !(e.f & 6144))) for (let e of n) {
				var r = e.f;
				if (r & 2) t(e);
				else {
					var i = e;
					r & 4194320 && !this.async_deriveds.has(i) && (this.#d.delete(i), j(i, g), this.schedule(i));
				}
			}
		};
		for (let e of this.current.keys()) t(e);
		this.oncommit(() => e.discard()), e.#S(), M = this, this.#_();
	}
	#x(e) {
		for (var t = 0; t < e.length; t += 1) Qe(e[t], this.#u, this.#d);
	}
	capture(e, t, n = !1) {
		e.v !== T && !this.previous.has(e) && this.previous.set(e, e.v), e.f & 8388608 || (this.current.set(e, [t, n]), yt?.set(e, t)), this.is_fork || (e.v = t);
	}
	activate() {
		M = this;
	}
	deactivate() {
		M = null, yt = null;
	}
	flush() {
		try {
			St = !0, M = this, this.#_();
		} finally {
			Tt = 0, bt = null, Ct = null, wt = null, St = !1, M = null, yt = null, It.clear();
		}
	}
	discard() {
		for (let e of this.#i) e(this);
		this.#i.clear();
		for (let e of this.async_deriveds.values()) e.reject(ct);
		this.#S(), this.#s?.resolve();
	}
	register_created_effect(e) {
		this.#l.push(e);
	}
	increment(e, t) {
		if (this.#a += 1, e) {
			let e = this.#o.get(t) ?? 0;
			this.#o.set(t, e + 1);
		}
	}
	decrement(e, t) {
		if (--this.#a, e) {
			let e = this.#o.get(t) ?? 0;
			e === 1 ? this.#o.delete(t) : this.#o.set(t, e - 1);
		}
		this.#m || (this.#m = !0, Je(() => {
			this.#m = !1, this.linked && this.flush();
		}));
	}
	transfer_effects(e, t) {
		for (let t of e) this.#u.add(t);
		for (let e of t) this.#d.add(e);
		e.clear(), t.clear();
	}
	oncommit(e) {
		this.#r.add(e);
	}
	ondiscard(e) {
		this.#i.add(e);
	}
	settled() {
		return (this.#s ??= m()).promise;
	}
	static ensure() {
		if (M === null) {
			let t = M = new e();
			!St && !xt && Je(() => {
				t.#e || t.flush();
			});
		}
		return M;
	}
	apply() {
		yt = null;
	}
	schedule(e) {
		if (bt = e, e.b?.is_pending && e.f & 16777228 && !(e.f & 32768)) {
			e.b.defer_effect(e);
			return;
		}
		this.#c.push(e);
	}
	#S() {
		if (this.linked) {
			var e = this.#t, t = this.#n;
			e === null || (e.#n = t), t === null ? _t = e : t.#t = e, this.linked = !1;
		}
	}
};
function Ot(e) {
	var t = xt;
	xt = !0;
	try {
		var n;
		for (e && (M !== null && !M.is_fork && M.flush(), n = e());;) {
			if (Ye(), M === null) return n;
			M.flush();
		}
	} finally {
		xt = t;
	}
}
function kt() {
	try {
		Pe();
	} catch (e) {
		cn(e, bt);
	}
}
var At = null;
function jt(e) {
	var t = e.length;
	if (t !== 0) {
		for (var n = 0; n < t;) {
			var r = e[n++];
			if (!(r.f & 24576) && Kn(r) && (At = /* @__PURE__ */ new Set(), Qn(r), r.deps === null && r.first === null && r.nodes === null && r.teardown === null && r.ac === null && En(r), At?.size > 0)) {
				It.clear();
				for (let e of At) {
					if (e.f & 24576) continue;
					let t = [e], n = e.parent;
					for (; n !== null;) At.has(n) && (At.delete(n), t.push(n)), n = n.parent;
					for (let e = t.length - 1; e >= 0; e--) {
						let n = t[e];
						n.f & 24576 || Qn(n);
					}
				}
				At.clear();
			}
		}
		At = null;
	}
}
function Mt(e) {
	M.schedule(e);
}
function Nt(e, t) {
	if (!(e.f & 32 && e.f & 1024)) {
		e.f & 2048 ? t.d.push(e) : e.f & 4096 && t.m.push(e), j(e, h);
		for (var n = e.first; n !== null;) Nt(n, t), n = n.next;
	}
}
function Pt(e) {
	j(e, h);
	for (var t = e.first; t !== null;) Pt(t), t = t.next;
}
//#endregion
//#region node_modules/svelte/src/internal/client/reactivity/sources.js
var Ft = /* @__PURE__ */ new Set(), It = /* @__PURE__ */ new Map(), Lt = !1;
function Rt(e, t) {
	return {
		f: 0,
		v: e,
		reactions: null,
		equals: Te,
		rv: 0,
		wv: 0
	};
}
/*#__NO_SIDE_EFFECTS__*/
function zt(e, t) {
	let n = Rt(e, t);
	return zn(n), n;
}
/*#__NO_SIDE_EFFECTS__*/
function N(e, t = !1, n = !0) {
	let r = Rt(e);
	return t || (r.equals = De), ze && n && A !== null && A.l !== null && (A.l.s ??= []).push(r), r;
}
function Bt(e, t) {
	return P(e, nr(() => Y(e))), t;
}
function P(e, t, n = !1) {
	return U !== null && (!In || U.f & 131072) && Ge() && U.f & 4325394 && (Rn === null || !Rn.has(e)) && Le(), Ut(e, n ? qt(t) : t, wt);
}
var Vt = null, Ht = 0;
function Ut(e, t, n = null) {
	if (!e.equals(t)) {
		Pn ? It.set(e, t) : It.has(e) || It.set(e, e.v);
		var r = Dt.ensure();
		if (r.capture(e, t), e.f & 2) {
			let t = e;
			e.f & 2048 && pt(t), yt === null && Ze(t);
		}
		e.wv = Gn(), Vt = null, Ht = 0, Kt(e, g, n), Vt = null, Ge() && G !== null && G.f & 1024 && !(G.f & 96) && (J === null ? Bn([e]) : J.push(e)), !r.is_fork && Ft.size > 0 && !Lt && Wt();
	}
	return t;
}
function Wt() {
	Lt = !1;
	for (let e of Ft) {
		e.f & 1024 && j(e, _);
		let t;
		try {
			t = Kn(e);
		} catch {
			t = !0;
		}
		t && Qn(e);
	}
	Ft.clear();
}
function Gt(e) {
	P(e, e.v + 1);
}
function Kt(e, t, n) {
	var r = e.reactions;
	if (r !== null) {
		var i = Ge(), a = r.length;
		if (Ht += a, Ht > 1e5 && Vt === null && (Vt = /* @__PURE__ */ new Set()), Vt !== null) {
			if (Vt.has(e)) return;
			Vt.add(e);
		}
		for (var o = 0; o < a; o++) {
			var s = r[o], c = s.f;
			if (i || s !== G) {
				var l = (c & g) === 0;
				if (l && j(s, t), c & 131072) Ft.add(s);
				else if (c & 2) {
					var u = s;
					yt?.delete(u), Kt(u, _, n);
				} else if (l) {
					var d = s;
					c & 16 && At !== null && At.add(d), n === null ? Mt(d) : n.push(d);
				}
			}
		}
	}
}
function qt(t) {
	if (typeof t != "object" || !t || ae in t || oe in t) return t;
	let n = l(t);
	if (n !== s && n !== c) return t;
	var r = /* @__PURE__ */ new Map(), i = e(t), o = /* @__PURE__ */ zt(0), u = null, d = Un, f = (e) => {
		if (Un === d) return e();
		var t = U, n = Un;
		W(null), Wn(d);
		var r = e();
		return W(t), Wn(n), r;
	};
	return i && r.set("length", /* @__PURE__ */ zt(t.length, u)), new Proxy(t, {
		defineProperty(e, t, n) {
			(!("value" in n) || n.configurable === !1 || n.enumerable === !1 || n.writable === !1) && Fe();
			var i = r.get(t);
			return i === void 0 ? f(() => {
				var e = /* @__PURE__ */ zt(n.value, u);
				return r.set(t, e), e;
			}) : P(i, n.value, !0), !0;
		},
		deleteProperty(e, t) {
			var n = r.get(t);
			if (n === void 0) {
				if (t in e) {
					let e = f(() => /* @__PURE__ */ zt(T, u));
					r.set(t, e), Gt(o);
				}
			} else P(n, T), Gt(o);
			return !0;
		},
		get(e, n, i) {
			if (n === ae) return t;
			var o = r.get(n), s = n in e;
			if (o === void 0 && (!s || a(e, n)?.writable) && (o = f(() => /* @__PURE__ */ zt(qt(s ? e[n] : T), u)), r.set(n, o)), o !== void 0) {
				var c = Y(o);
				return c === T ? void 0 : c;
			}
			return Reflect.get(e, n, i);
		},
		getOwnPropertyDescriptor(e, t) {
			this.has?.(e, t);
			var n = Reflect.getOwnPropertyDescriptor(e, t), i = r.get(t);
			if (i !== void 0) {
				var a = Y(i);
				if (a === T) return;
				if (n && "value" in n) n.value = a;
				else return {
					enumerable: !0,
					configurable: !0,
					value: a,
					writable: !0
				};
			}
			return n;
		},
		has(e, t) {
			if (t === ae) return !0;
			var n = r.get(t), i = n !== void 0 && n.v !== T || Reflect.has(e, t);
			return (n !== void 0 || G !== null && (!i || a(e, t)?.writable)) && (n === void 0 && (n = f(() => /* @__PURE__ */ zt(i ? qt(e[t]) : T, u)), r.set(t, n)), Y(n) === T) ? !1 : i;
		},
		set(e, t, n, s) {
			var c = r.get(t), l = t in e;
			if (i && t === "length") for (var d = n; d < c.v; d += 1) {
				var p = r.get(d + "");
				p === void 0 ? d in e && (p = f(() => /* @__PURE__ */ zt(T, u)), r.set(d + "", p)) : P(p, T);
			}
			if (c === void 0) (!l || a(e, t)?.writable) && (c = f(() => /* @__PURE__ */ zt(void 0, u)), P(c, qt(n)), r.set(t, c));
			else {
				l = c.v !== T;
				var m = f(() => qt(n));
				P(c, m);
			}
			var h = Reflect.getOwnPropertyDescriptor(e, t);
			if (h?.set && h.set.call(s, n), !l) {
				if (i && typeof t == "string") {
					var g = r.get("length"), _ = Number(t);
					Number.isInteger(_) && _ >= g.v && P(g, _ + 1);
				}
				Gt(o);
			}
			return !0;
		},
		ownKeys(e) {
			Y(o);
			var t = Reflect.ownKeys(e).filter((e) => {
				var t = r.get(e);
				return t === void 0 || t.v !== T;
			});
			for (var [n, i] of r) i.v !== T && !(n in e) && t.push(n);
			return t;
		},
		setPrototypeOf() {
			Ie();
		}
	});
}
var Jt, Yt, Xt, Zt, Qt;
function $t() {
	if (Jt === void 0) {
		Jt = window, Yt = document, Xt = /Firefox/.test(navigator.userAgent);
		var e = Element.prototype, t = Node.prototype, n = Text.prototype;
		Zt = a(t, "firstChild").get, Qt = a(t, "nextSibling").get, u(e) && (e[le] = void 0, e[ce] = null, e[ue] = void 0, e.__e = void 0), u(n) && (n[de] = void 0);
	}
}
function F(e = "") {
	return document.createTextNode(e);
}
/*@__NO_SIDE_EFFECTS__*/
function en(e) {
	return Zt.call(e);
}
/*@__NO_SIDE_EFFECTS__*/
function tn(e) {
	return Qt.call(e);
}
function I(e, t) {
	if (!E) return /* @__PURE__ */ en(e);
	var n = /* @__PURE__ */ en(D);
	if (n === null) n = D.appendChild(F());
	else if (t && n.nodeType !== 3) {
		var r = F();
		return n?.before(r), O(r), r;
	}
	return t && on(n), O(n), n;
}
function L(e, t = !1) {
	if (!E) {
		var n = /* @__PURE__ */ en(e);
		return n instanceof Comment && n.data === "" ? /* @__PURE__ */ tn(n) : n;
	}
	if (t) {
		if (D?.nodeType !== 3) {
			var r = F();
			return D?.before(r), O(r), r;
		}
		on(D);
	}
	return D;
}
function R(e, t = !1) {
	if (!E) return /* @__PURE__ */ en(e);
	var n = I(e, t);
	return k(e), n;
}
function z(e, t = 1, n = !1) {
	let r = E ? D : e;
	for (var i; t--;) i = r, r = /* @__PURE__ */ tn(r);
	if (!E) return r;
	if (n) {
		if (r?.nodeType !== 3) {
			var a = F();
			return r === null ? i?.after(a) : r.before(a), O(a), a;
		}
		on(r);
	}
	return O(r), r;
}
function nn(e) {
	e.textContent = "";
}
function rn() {
	return !1;
}
function an(e, t, n) {
	return t == null || t === "http://www.w3.org/1999/xhtml" ? n ? document.createElement(e, { is: n }) : document.createElement(e) : n ? document.createElementNS(t, e, { is: n }) : document.createElementNS(t, e);
}
function on(e) {
	if (e.nodeValue.length < 65536) return;
	let t = e.nextSibling;
	for (; t !== null && t.nodeType === 3;) t.remove(), e.nodeValue += t.nodeValue, t = e.nextSibling;
}
function sn(e) {
	var t = G;
	if (t === null) return U.f |= w, e;
	if (!(t.f & 32768) && !(t.f & 4)) throw e;
	cn(e, t);
}
function cn(e, t) {
	if (!(t !== null && t.f & 16384)) {
		for (; t !== null;) {
			if (t.f & 128 && !(t.f & 33570816)) {
				if (!(t.f & 32768)) throw e;
				try {
					t.b.error(e);
					return;
				} catch (t) {
					e = t;
				}
			}
			t = t.parent;
		}
		throw e;
	}
}
//#endregion
//#region node_modules/svelte/src/internal/client/reactivity/effects.js
function ln(e) {
	G === null && (U === null && Ne(e), Me()), Pn && je(e);
}
function un(e, t) {
	var n = t.last;
	n === null ? t.last = t.first = e : (n.next = e, e.prev = n, t.last = e);
}
function dn(e, t) {
	var n = G;
	n !== null && n.f & 8192 && (e |= v);
	var r = {
		ctx: A,
		deps: null,
		nodes: null,
		f: e | g | 512,
		first: null,
		fn: t,
		last: null,
		next: null,
		parent: n,
		b: n && n.b,
		prev: null,
		teardown: null,
		wv: 0,
		ac: null
	};
	M?.register_created_effect(r);
	var i = r;
	if (e & 4) Ct === null ? Dt.ensure().schedule(r) : Ct.push(r);
	else if (t !== null) {
		try {
			Qn(r);
		} catch (e) {
			throw H(r), e;
		}
		i.deps === null && i.teardown === null && i.nodes === null && i.first === i.last && !(i.f & 524288) && (i = i.first, e & 16 && e & 65536 && i !== null && (i.f |= S));
	}
	if (i !== null && (i.parent = n, n !== null && un(i, n), U !== null && U.f & 2 && !(e & 64))) {
		var a = U;
		(a.effects ??= []).push(i);
	}
	return r;
}
function fn() {
	return U !== null && !In;
}
function pn(e) {
	let t = dn(8, null);
	return j(t, h), t.teardown = e, t;
}
function mn(e) {
	ln("$effect");
	var t = G.f;
	if (!U && t & 32 && A !== null && !A.i) {
		var n = A;
		(n.e ??= []).push(e);
	} else return hn(e);
}
function hn(e) {
	return dn(4 | te, e);
}
function gn(e) {
	return ln("$effect.pre"), dn(8 | te, e);
}
function _n(e) {
	Dt.ensure();
	let t = dn(64 | C, e);
	return (e = {}) => new Promise((n) => {
		e.outro ? Dn(t, () => {
			H(t), n(void 0);
		}) : (H(t), n(void 0));
	});
}
function vn(e) {
	return dn(4, e);
}
function yn(e) {
	return dn(ie | C, e);
}
function bn(e, t = 0) {
	return dn(8 | t, e);
}
function B(e, t = [], n = [], r = []) {
	rt(r, t, n, (t) => {
		dn(8, () => {
			e(...t.map(Y));
		});
	});
}
function xn(e, t = 0) {
	return dn(16 | t, e);
}
function V(e) {
	return dn(32 | C, e);
}
function Sn(e) {
	var t = e.teardown;
	if (t !== null) {
		let n = Pn, r = U;
		Fn(!0), W(null);
		try {
			t.call(null);
		} catch (t) {
			cn(t, e.parent);
		} finally {
			Fn(n), W(r);
		}
	}
}
function Cn(e, t = !1) {
	var n = e.first;
	for (e.first = e.last = null; n !== null;) {
		let e = n.ac;
		e !== null && tt(() => {
			e.abort(pe);
		});
		var r = n.next;
		n.f & 64 ? n.parent = null : H(n, t), n = r;
	}
}
function wn(e) {
	for (var t = e.first; t !== null;) {
		var n = t.next;
		t.f & 32 || H(t), t = n;
	}
}
function H(e, t = !0) {
	var n = !1;
	(t || e.f & 262144) && e.nodes !== null && e.nodes.end !== null && (Tn(e.nodes.start, e.nodes.end), n = !0), e.f |= x, Cn(e, t && !n), Zn(e, 0);
	var r = e.nodes && e.nodes.t;
	if (r !== null) for (let e of r) e.stop();
	Sn(e), e.f ^= x, e.f |= y;
	var i = e.parent;
	i !== null && i.first !== null && En(e), e.next = e.prev = e.teardown = e.ctx = e.deps = e.fn = e.nodes = e.ac = e.b = null;
}
function Tn(e, t) {
	for (; e !== null;) {
		var n = e === t ? null : /* @__PURE__ */ tn(e);
		e.remove(), e = n;
	}
}
function En(e) {
	var t = e.parent, n = e.prev, r = e.next;
	n !== null && (n.next = r), r !== null && (r.prev = n), t !== null && (t.first === e && (t.first = r), t.last === e && (t.last = n));
}
function Dn(e, t, n = !0) {
	var r = [];
	e.f |= 256, On(e, r, !0);
	var i = () => {
		n && H(e), t && t();
	}, a = r.length;
	if (a > 0) {
		var o = () => --a || i();
		for (var s of r) s.out(o);
	} else i();
}
function On(e, t, n) {
	if (!(e.f & 8192)) {
		e.f ^= v;
		var r = e.nodes && e.nodes.t;
		if (r !== null) for (let e of r) (e.is_global || n) && t.push(e);
		for (var i = e.first; i !== null;) {
			var a = i.next;
			if (!(i.f & 64)) {
				var o = !!(i.f & 65536) || !!(i.f & 32) && !!(e.f & 16);
				On(i, t, o ? n : !1);
			}
			i = a;
		}
	}
}
function kn(e) {
	e.f &= -257, An(e, !0);
}
function An(e, t) {
	if (!(e.f & 256) && e.f & 8192) {
		e.f ^= v, e.f & 1024 || (j(e, g), Dt.ensure().schedule(e));
		for (var n = e.first; n !== null;) {
			var r = n.next, i = !!(n.f & 65536) || !!(n.f & 32);
			An(n, i ? t : !1), n = r;
		}
		var a = e.nodes && e.nodes.t;
		if (a !== null) for (let e of a) (e.is_global || t) && e.in();
	}
}
function jn(e, t) {
	if (e.nodes) for (var n = e.nodes.start, r = e.nodes.end; n !== null;) {
		var i = n === r ? null : /* @__PURE__ */ tn(n);
		t.append(n), n = i;
	}
}
//#endregion
//#region node_modules/svelte/src/internal/client/legacy.js
var Mn = null, Nn = !1, Pn = !1;
function Fn(e) {
	Pn = e;
}
var U = null, In = !1;
function W(e) {
	U = e;
}
var G = null;
function Ln(e) {
	G = e;
}
var Rn = null;
function zn(e) {
	U !== null && (U.f & 2097152 || U.f & 2) && (Rn ??= /* @__PURE__ */ new Set()).add(e);
}
var K = null, q = 0, J = null;
function Bn(e) {
	J = e;
}
var Vn = 1, Hn = 0, Un = Hn;
function Wn(e) {
	Un = e;
}
function Gn() {
	return ++Vn;
}
function Kn(e) {
	var t = e.f;
	if (t & 2048) return !0;
	if (t & 4096) {
		for (var n = e.deps, r = n.length, i = 0; i < r; i++) {
			var a = n[i];
			if (Kn(a) && mt(a), a.wv > e.wv) return !0;
		}
		t & 512 && yt === null && j(e, h);
	}
	return !1;
}
function qn(e, t, n = !0) {
	var r = e.reactions;
	if (r !== null && !(Rn !== null && Rn.has(e))) for (var i = 0; i < r.length; i++) {
		var a = r[i];
		a.f & 2 ? qn(a, t, !1) : t === a && (n ? j(a, g) : a.f & 1024 && j(a, _), Mt(a));
	}
}
function Jn(e) {
	var t = K, n = q, r = J, i = U, a = Rn, o = A, s = In, c = Un, l = e.f;
	K = null, q = 0, J = null, U = l & 96 ? null : e, Rn = null, Ve(e.ctx), In = !1, Un = ++Hn, e.ac !== null && (tt(() => {
		e.ac.abort(pe);
	}), e.ac = null);
	try {
		e.f |= re;
		var u = e.fn, d = u();
		e.f |= b;
		var f = Yn(e);
		if (Ge() && J !== null && !In && f !== null && !(e.f & 6146)) for (var p = 0; p < J.length; p++) qn(J[p], e);
		if (i !== null && i !== e) {
			if (Hn++, i.deps !== null) for (let e = 0; e < n; e += 1) i.deps[e].rv = Hn;
			if (t !== null) for (let e of t) e.rv = Hn;
			J !== null && (r === null ? r = J : r.push(...J));
		}
		return e.f & 8388608 && (e.f ^= w), d;
	} catch (t) {
		return Yn(e), sn(t);
	} finally {
		e.f ^= re, K = t, q = n, J = r, U = i, Rn = a, Ve(o), In = s, Un = c;
	}
}
function Yn(e) {
	var t = e.deps, n = M?.is_fork;
	if (K !== null) {
		var r;
		if (n || Zn(e, q), t !== null && q > 0) for (t.length = q + K.length, r = 0; r < K.length; r++) t[q + r] = K[r];
		else e.deps = t = K;
		if (fn() && e.f & 512) for (r = q; r < t.length; r++) (t[r].reactions ??= []).push(e);
	} else !n && t !== null && q < t.length && (Zn(e, q), t.length = q);
	return t;
}
function Xn(e, r) {
	let i = r.reactions;
	if (i !== null) {
		var a = t.call(i, e);
		if (a !== -1) {
			var o = i.length - 1;
			o === 0 ? i = r.reactions = null : (i[a] = i[o], i.pop());
		}
	}
	if (i === null && r.f & 2 && (K === null || !n.call(K, r))) {
		var s = r;
		s.f & 512 && (s.f ^= 512), s.v !== T && Ze(s), s.ac !== null && tt(() => {
			s.ac.abort(pe), s.ac = null, j(s, g);
		}), ht(s), Zn(s, 0);
	}
}
function Zn(e, t) {
	var n = e.deps;
	if (n !== null) for (var r = t; r < n.length; r++) Xn(e, n[r]);
}
function Qn(e) {
	var t = e.f;
	if (!(t & 16384)) {
		j(e, h);
		var n = G, r = Nn;
		G = e, Nn = !(t & 96);
		try {
			t & 16777232 ? wn(e) : Cn(e), Sn(e);
			var i = Jn(e);
			e.teardown = typeof i == "function" ? i : null, e.wv = Vn;
		} finally {
			Nn = r, G = n;
		}
	}
}
async function $n() {
	await Promise.resolve(), Ot();
}
function Y(e) {
	var t = !!(e.f & 2);
	if (Mn?.add(e), U !== null && !In && !(G !== null && G.f & 16384) && (Rn === null || !Rn.has(e))) {
		var r = U.deps;
		if (U.f & 2097152) e.rv < Hn && (e.rv = Hn, K === null && r !== null && r[q] === e ? q++ : K === null ? K = [e] : K.push(e));
		else {
			U.deps ??= [], n.call(U.deps, e) || U.deps.push(e);
			var i = e.reactions;
			i === null ? e.reactions = [U] : n.call(i, U) || i.push(U);
		}
	}
	if (Pn && It.has(e)) return It.get(e);
	if (t) {
		var a = e;
		if (Pn) {
			var o = a.v;
			return (!(a.f & 1024) && a.reactions !== null || tr(a)) && (o = pt(a)), It.set(a, o), o;
		}
		var s = !(a.f & 512) && !In && U !== null && (Nn || !!(U.f & 512)), c = (a.f & b) === 0;
		Kn(a) && (s && (a.f |= 512), mt(a)), s && !c && (gt(a), er(a));
	}
	if (yt?.has(e)) return yt.get(e);
	if (e.f & 8388608) throw e.v;
	return e.v;
}
function er(e) {
	if (e.f |= 512, e.deps !== null) for (let t of e.deps) (t.reactions ??= []).push(e), t.f & 2 && !(t.f & 512) && (gt(t), er(t));
}
function tr(e) {
	if (e.v === T) return !0;
	if (e.deps === null) return !1;
	for (let t of e.deps) if (It.has(t) || t.f & 2 && tr(t)) return !0;
	return !1;
}
function nr(e) {
	var t = In;
	try {
		return In = !0, e();
	} finally {
		In = t;
	}
}
function rr(e) {
	if (!(typeof e != "object" || !e || e instanceof EventTarget)) {
		if (ae in e) ir(e);
		else if (!Array.isArray(e)) for (let t in e) {
			let n = e[t];
			typeof n == "object" && n && ae in n && ir(n);
		}
	}
}
function ir(e, t = /* @__PURE__ */ new Set()) {
	if (typeof e == "object" && e && !(e instanceof EventTarget) && !t.has(e)) {
		t.add(e), e instanceof Date && e.getTime();
		for (let n in e) try {
			ir(e[n], t);
		} catch {}
		let n = l(e);
		if (n !== Object.prototype && n !== Array.prototype && n !== Map.prototype && n !== Set.prototype && n !== Date.prototype) {
			let t = o(n);
			for (let n in t) {
				let r = t[n].get;
				if (r) try {
					r.call(e);
				} catch {}
			}
		}
	}
}
[.../* @__PURE__ */ "allowfullscreen.async.autofocus.autoplay.checked.controls.default.disabled.formnovalidate.indeterminate.inert.ismap.loop.multiple.muted.nomodule.novalidate.open.playsinline.readonly.required.reversed.seamless.selected.webkitdirectory.defer.disablepictureinpicture.disableremoteplayback".split(".")];
var ar = ["touchstart", "touchmove"];
function or(e) {
	return ar.includes(e);
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/elements/events.js
var sr = Symbol("events"), cr = /* @__PURE__ */ new Set(), lr = /* @__PURE__ */ new Set();
function ur(e, t, n, r = {}) {
	function i(e) {
		if (r.capture || pr.call(t, e), !e.cancelBubble) return tt(() => n?.call(this, e));
	}
	return e.startsWith("pointer") || e.startsWith("touch") || e === "wheel" ? (i.__removed = !1, Je(() => {
		i.__removed || t.addEventListener(e, i, r);
	})) : t.addEventListener(e, i, r), i;
}
function X(e, t, n, r, i) {
	var a = {
		capture: r,
		passive: i
	}, o = ur(e, t, n, a);
	(t === document.body || t === window || t === document || t instanceof HTMLMediaElement) && pn(() => {
		o.__removed = !0, t.removeEventListener(e, o, a);
	});
}
var dr = null, fr = !1;
function pr(e) {
	var t = this, n = t.ownerDocument, r = e.type, a = e.composedPath?.() || [], o = a[0] || e.target;
	dr = e, fr || (fr = !0, setTimeout(() => {
		fr = !1, dr = null;
	}));
	var s = 0, c = dr === e && e[sr];
	if (c) {
		var l = a.indexOf(c);
		if (l !== -1 && (t === document || t === window)) {
			e[sr] = t;
			return;
		}
		var u = a.indexOf(t);
		if (u === -1) return;
		l <= u && (s = l);
	}
	if (o = a[s] || e.target, o !== t) {
		i(e, "currentTarget", {
			configurable: !0,
			get() {
				return o || n;
			}
		});
		var d = U, f = G;
		W(null), Ln(null);
		try {
			for (var p, m = []; o !== null && o !== t;) {
				try {
					var h = o[sr]?.[r];
					h != null && (!o.disabled || e.target === o) && h.call(o, e);
				} catch (e) {
					p ? m.push(e) : p = e;
				}
				if (e.cancelBubble) break;
				s++, o = s < a.length ? a[s] : null;
			}
			if (p) {
				for (let e of m) queueMicrotask(() => {
					throw e;
				});
				throw p;
			}
		} finally {
			e[sr] = t, delete e.currentTarget, W(d), Ln(f);
		}
	}
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/reconciler.js
var mr = globalThis?.window?.trustedTypes && /* @__PURE__ */ globalThis.window.trustedTypes.createPolicy("svelte-trusted-html", { createHTML: (e) => e });
function hr(e) {
	return mr?.createHTML(e) ?? e;
}
function gr(e) {
	var t = an("template");
	return t.innerHTML = hr(e.replaceAll("<!>", "<!---->")), t.content;
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/template.js
function _r(e, t) {
	var n = G;
	n.nodes === null && (n.nodes = {
		start: e,
		end: t,
		a: null,
		t: null
	});
}
/*#__NO_SIDE_EFFECTS__*/
function Z(e, t) {
	var n = !!(t & 1), r = !!(t & 2), i, a = !e.startsWith("<!>");
	return () => {
		if (E) return _r(D, null), D;
		i === void 0 && (i = gr(a ? e : "<!>" + e), n || (i = /* @__PURE__ */ en(i)));
		var t = r || Xt ? document.importNode(i, !0) : i.cloneNode(!0);
		if (n) {
			var o = /* @__PURE__ */ en(t), s = t.lastChild;
			_r(o, s);
		} else _r(t, t);
		return t;
	};
}
function Q(e, t) {
	if (E) {
		var n = G;
		(!(n.f & 32768) || n.nodes.end === null) && (n.nodes.end = D), xe();
		return;
	}
	e !== null && e.before(t);
}
//#endregion
//#region node_modules/svelte/src/reactivity/create-subscriber.js
function vr(e) {
	let t = 0, n = Rt(0), r;
	return () => {
		fn() && (Y(n), bn(() => (t === 0 && (r = nr(() => e(() => Gt(n)))), t += 1, () => {
			Je(() => {
				--t, t === 0 && (r?.(), r = void 0, Gt(n));
			});
		})));
	};
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/blocks/boundary.js
var yr = S | C;
function br(e, t, n, r) {
	new xr(e, t, n, r);
}
var xr = class {
	parent;
	is_pending = !1;
	transform_error;
	#e;
	#t = E ? D : null;
	#n;
	#r;
	#i;
	#a = null;
	#o = null;
	#s = null;
	#c = null;
	#l = 0;
	#u = 0;
	#d = !1;
	#f = /* @__PURE__ */ new Set();
	#p = /* @__PURE__ */ new Set();
	#m = null;
	#h = vr(() => (this.#m = Rt(this.#l), () => {
		this.#m = null;
	}));
	constructor(e, t, n, r) {
		this.#e = e, this.#n = t, this.#r = (e) => {
			var t = G;
			t.b = this, t.f |= 128, n(e);
		}, this.parent = G.b, this.transform_error = r ?? this.parent?.transform_error ?? ((e) => e), this.#i = xn(() => {
			if (E) {
				let e = this.#t;
				xe();
				let t = e.data === "[!";
				if (e.data.startsWith("[?")) {
					let t = JSON.parse(e.data.slice(2));
					this.#_(t);
				} else t ? this.#y() : this.#g();
			} else this.#b();
		}, yr), E && (this.#e = D);
	}
	#g() {
		try {
			this.#a = V(() => this.#r(this.#e));
		} catch (e) {
			this.error(e);
		}
	}
	#_(e) {
		let t = this.#n.failed, { reset: n, invoke_onerror: r } = this.#v(e);
		Je(r), t && (this.#s = V(() => {
			t(this.#e, () => e, () => n);
		}));
	}
	#v(e) {
		var t = !1, n = !1;
		let r = () => {
			if (t) {
				ye();
				return;
			}
			t = !0, n && Re(), this.#s !== null && Dn(this.#s, () => {
				this.#s = null;
			}), this.#S(() => {
				this.#b();
			});
		};
		return {
			reset: r,
			invoke_onerror: () => {
				try {
					n = !0, this.#n.onerror?.(e, r), n = !1;
				} catch (e) {
					cn(e, this.#i && this.#i.parent);
				}
			}
		};
	}
	#y() {
		let e = this.#n.pending;
		e && (this.is_pending = !0, this.#o = V(() => e(this.#e)), Je(() => {
			var e = this.#c = document.createDocumentFragment(), t = F(), n = !1;
			if (e.append(t), this.#a = this.#S(() => {
				try {
					return V(() => this.#r(t));
				} catch (e) {
					try {
						this.error(e), n = !0;
					} catch (e) {
						cn(e, this.#i.parent);
					}
					return null;
				}
			}), this.#a === null) {
				this.#c = null, n && this.#x(M);
				return;
			}
			this.#u === 0 && (this.#e.before(e), this.#c = null, Dn(this.#o, () => {
				this.#o = null;
			}), this.#x(M));
		}));
	}
	#b() {
		try {
			if (this.is_pending = this.has_pending_snippet(), this.#u = 0, this.#l = 0, this.#a = V(() => {
				this.#r(this.#e);
			}), this.#u > 0) {
				var e = this.#c = document.createDocumentFragment();
				jn(this.#a, e);
				let t = this.#n.pending;
				this.#o = V(() => t(this.#e));
			} else this.#x(M);
		} catch (e) {
			this.error(e);
		}
	}
	#x(e) {
		this.is_pending = !1, e.transfer_effects(this.#f, this.#p);
	}
	defer_effect(e) {
		Qe(e, this.#f, this.#p);
	}
	is_rendered() {
		return !this.is_pending && (!this.parent || this.parent.is_rendered());
	}
	has_pending_snippet() {
		return !!this.#n.pending;
	}
	#S(e) {
		var t = G, n = U, r = A;
		Ln(this.#i), W(this.#i), Ve(this.#i.ctx);
		try {
			return Dt.ensure(), e();
		} finally {
			Ln(t), W(n), Ve(r);
		}
	}
	#C(e, t) {
		if (!this.has_pending_snippet()) {
			this.parent && this.parent.#C(e, t);
			return;
		}
		this.#u += e, this.#u === 0 && (this.#x(t), this.#o && Dn(this.#o, () => {
			this.#o = null;
		}), this.#c &&= (this.#e.before(this.#c), null));
	}
	update_pending_count(e, t) {
		this.#C(e, t), this.#l += e, !(!this.#m || this.#d) && (this.#d = !0, Je(() => {
			this.#d = !1, this.#m && Ut(this.#m, this.#l);
		}));
	}
	get_effect_pending() {
		return this.#h(), Y(this.#m);
	}
	error(e) {
		if (!this.#n.onerror && !this.#n.failed) throw e;
		M?.is_fork ? (this.#a && M.skip_effect(this.#a), this.#o && M.skip_effect(this.#o), this.#s && M.skip_effect(this.#s), M.oncommit(() => {
			this.#w(e);
		})) : this.#w(e);
	}
	#w(e) {
		this.#a &&= (H(this.#a), null), this.#o &&= (H(this.#o), null), this.#s &&= (H(this.#s), null), E && (O(this.#t), Se(), O(Ce()));
		let t = this.#n.failed, n = (e) => {
			let { reset: n, invoke_onerror: r } = this.#v(e);
			r(), t && (this.#s = this.#S(() => {
				try {
					return V(() => {
						var r = G;
						r.b = this, r.f |= 128, t(this.#e, () => e, () => n);
					});
				} catch (e) {
					return cn(e, this.#i.parent), null;
				}
			}));
		};
		Je(() => {
			var t;
			try {
				t = this.transform_error(e);
			} catch (e) {
				cn(e, this.#i && this.#i.parent);
				return;
			}
			typeof t == "object" && t && typeof t.then == "function" ? t.then(n, (e) => cn(e, this.#i && this.#i.parent)) : n(t);
		});
	}
};
function $(e, t) {
	var n = t == null ? "" : typeof t == "object" ? `${t}` : t;
	n !== (e[de] ??= e.nodeValue) && (e[de] = n, e.nodeValue = `${n}`);
}
function Sr(e, t) {
	return wr(e, t);
}
var Cr = /* @__PURE__ */ new Map();
function wr(e, { target: t, anchor: n, props: i = {}, events: a, context: o, intro: s = !0, transformError: c }) {
	$t();
	var l = void 0, u = _n(() => {
		var s = n ?? t.appendChild(F());
		br(s, { pending: () => {} }, (t) => {
			He({});
			var n = A;
			if (o && (n.c = o), a && (i.$$events = a), E && _r(t, null), l = e(t, i) || We(), E && (G.nodes.end = D, D === null || D.nodeType !== 8 || D.data !== "]")) throw ve(), he;
			Ue();
		}, c);
		var u = /* @__PURE__ */ new Set(), d = (e) => {
			for (var n = 0; n < e.length; n++) {
				var r = e[n];
				if (!u.has(r)) {
					u.add(r);
					var i = or(r);
					for (let e of [t, document]) {
						var a = Cr.get(e);
						a === void 0 && (a = /* @__PURE__ */ new Map(), Cr.set(e, a));
						var o = a.get(r);
						o === void 0 ? (e.addEventListener(r, pr, { passive: i }), a.set(r, 1)) : a.set(r, o + 1);
					}
				}
			}
		};
		return d(r(cr)), lr.add(d), () => {
			for (var e of u) for (let n of [t, document]) {
				var r = Cr.get(n), i = r.get(e);
				--i == 0 ? (n.removeEventListener(e, pr), r.delete(e), r.size === 0 && Cr.delete(n)) : r.set(e, i);
			}
			lr.delete(d), s !== n && s.parentNode?.removeChild(s);
		};
	});
	return Tr.set(l, u), l;
}
var Tr = /* @__PURE__ */ new WeakMap(), Er = class {
	anchor;
	#e = /* @__PURE__ */ new Map();
	#t = /* @__PURE__ */ new Map();
	#n = /* @__PURE__ */ new Map();
	#r = /* @__PURE__ */ new Set();
	#i = !0;
	constructor(e, t = !0) {
		this.anchor = e, this.#i = t;
	}
	#a = (e) => {
		if (this.#e.has(e)) {
			var t = this.#e.get(e), n = this.#t.get(t);
			if (n) kn(n), this.#r.delete(t);
			else {
				var r = this.#n.get(t);
				r && (kn(r.effect), this.#t.set(t, r.effect), this.#n.delete(t), r.fragment.lastChild.remove(), this.anchor.before(r.fragment), n = r.effect);
			}
			for (let [t, n] of this.#e) {
				if (this.#e.delete(t), t === e) break;
				let r = this.#n.get(n);
				r && (H(r.effect), this.#n.delete(n));
			}
			for (let [e, r] of this.#t) {
				if (e === t || this.#r.has(e)) continue;
				let i = () => {
					if (Array.from(this.#e.values()).includes(e)) {
						var t = document.createDocumentFragment();
						jn(r, t), t.append(F()), this.#n.set(e, {
							effect: r,
							fragment: t
						});
					} else H(r);
					this.#r.delete(e), this.#t.delete(e);
				};
				this.#i || !n ? (this.#r.add(e), Dn(r, i, !1)) : i();
			}
		}
	};
	#o = (e) => {
		this.#e.delete(e);
		let t = Array.from(this.#e.values());
		for (let [e, n] of this.#n) t.includes(e) || (H(n.effect), this.#n.delete(e));
	};
	ensure(e, t) {
		var n = M, r = rn();
		if (t && !this.#t.has(e) && !this.#n.has(e)) {
			if (r) {
				var i = document.createDocumentFragment(), a = F();
				i.append(a), this.#n.set(e, {
					effect: V(() => t(a)),
					fragment: i
				});
			} else this.#t.set(e, V(() => t(this.anchor)));
		}
		if (this.#e.set(n, e), r) {
			for (let [t, r] of this.#t) t === e ? n.unskip_effect(r) : n.skip_effect(r);
			for (let [t, r] of this.#n) t === e ? n.unskip_effect(r.effect) : n.skip_effect(r.effect);
			n.oncommit(this.#a), n.ondiscard(this.#o);
		} else E && (this.anchor = D), this.#a(n);
	}
};
//#endregion
//#region node_modules/svelte/src/internal/client/dom/blocks/if.js
function Dr(e, t, n = !1) {
	var r;
	E && (r = D, xe());
	var i = new Er(e), a = n ? S : 0;
	function o(e, t) {
		if (E) {
			var n = we(r);
			if (e !== parseInt(n.substring(1))) {
				var a = Ce();
				O(a), i.anchor = a, be(!1), i.ensure(e, t), be(!0);
				return;
			}
		}
		i.ensure(e, t);
	}
	xn(() => {
		var e = !1;
		t((t, n = 0) => {
			e = !0, o(n, t);
		}), e || o(-1, null);
	}, a);
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/blocks/key.js
var Or = Symbol("NaN");
function kr(e, t, n) {
	E && xe();
	var r = new Er(e), i = !Ge();
	xn(() => {
		var e = t();
		e !== e && (e = Or), i && typeof e == "object" && e && (e = {}), r.ensure(e, n);
	});
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/blocks/each.js
function Ar(e, t) {
	return t;
}
function jr(e, t, n) {
	for (var i = [], a = t.length, o, s = t.length, c = 0; c < a; c++) {
		let n = t[c];
		Dn(n, () => {
			if (o) {
				if (o.pending.delete(n), o.done.add(n), o.pending.size === 0) {
					var t = e.outrogroups;
					Mr(e, r(o.done)), t.delete(o), t.size === 0 && (e.outrogroups = null);
				}
			} else --s;
		}, !1);
	}
	if (s === 0) {
		var l = i.length === 0 && n !== null && e.pending.size === 0;
		if (l) {
			var u = n, d = u.parentNode;
			nn(d), d.append(u), e.items.clear();
		}
		Mr(e, t, !l);
	} else o = {
		pending: new Set(t),
		done: /* @__PURE__ */ new Set()
	}, (e.outrogroups ??= /* @__PURE__ */ new Set()).add(o);
}
function Mr(e, t, n = !0) {
	var r;
	if (e.pending.size > 0) {
		r = /* @__PURE__ */ new Set();
		for (let t of e.pending.values()) for (let n of t) r.add(e.items.get(n).e);
	}
	for (var i = 0; i < t.length; i++) {
		var a = t[i];
		r?.has(a) ? (a.f |= ne, jn(a, document.createDocumentFragment())) : H(t[i], n);
	}
}
var Nr;
function Pr(t, n, i, a, o, s = null) {
	var c = t, l = /* @__PURE__ */ new Map();
	if (n & 4) {
		var u = t;
		c = E ? O(/* @__PURE__ */ en(u)) : u.appendChild(F());
	}
	E && xe();
	var d = null, f = /* @__PURE__ */ dt(() => {
		var t = i();
		return e(t) ? t : t == null ? [] : r(t);
	}), p, m = /* @__PURE__ */ new Map(), h = !0;
	function g(e) {
		v.effect.f & 16384 || (v.pending.delete(e), v.fallback = d, Ir(v, p, c, n, a), d !== null && (p.length === 0 ? d.f & 33554432 ? (d.f ^= ne, Rr(d, null, c)) : kn(d) : Dn(d, () => {
			d = null;
		})));
	}
	function _(e) {
		v.pending.delete(e);
	}
	var v = {
		effect: xn(() => {
			p = Y(f);
			var e = p.length;
			let t = !1;
			E && we(c) === "[!" != (e === 0) && (c = Ce(), O(c), be(!1), t = !0);
			for (var r = /* @__PURE__ */ new Set(), u = M, v = rn(), y = 0; y < e; y += 1) {
				E && D.nodeType === 8 && D.data === "]" && (c = D, t = !0, be(!1));
				var b = p[y], x = a(b, y), S = h ? null : l.get(x);
				S ? (S.v && Ut(S.v, b), S.i && Ut(S.i, y), v && u.unskip_effect(S.e)) : (S = Lr(l, h ? c : Nr ??= F(), b, x, y, o, n, i), h || (S.e.f |= ne), l.set(x, S)), r.add(x);
			}
			if (e === 0 && s && !d && (h ? d = V(() => s(c)) : (d = V(() => s(Nr ??= F())), d.f |= ne)), e > r.size && Ae("", "", ""), E && e > 0 && O(Ce()), !h) {
				if (m.set(u, r), v) {
					for (let [e, t] of l) r.has(e) || u.skip_effect(t.e);
					u.oncommit(g), u.ondiscard(_);
				} else g(u);
			}
			t && be(!0), Y(f);
		}),
		flags: n,
		items: l,
		pending: m,
		outrogroups: null,
		fallback: d
	};
	h = !1, E && (c = D);
}
function Fr(e) {
	for (; e !== null && !(e.f & 32);) e = e.next;
	return e;
}
function Ir(e, t, n, i, a) {
	var o = !!(i & 8), s = t.length, c = e.items, l = Fr(e.effect.first), u, d = null, f, p = [], m = [], h, g, _, v;
	if (o) for (v = 0; v < s; v += 1) h = t[v], g = a(h, v), _ = c.get(g).e, _.f & 33554432 || (_.nodes?.a?.measure(), (f ??= /* @__PURE__ */ new Set()).add(_));
	for (v = 0; v < s; v += 1) {
		if (h = t[v], g = a(h, v), _ = c.get(g).e, e.outrogroups !== null) for (let t of e.outrogroups) t.pending.delete(_), t.done.delete(_);
		if (_.f & 8192 && (kn(_), o && (_.nodes?.a?.unfix(), (f ??= /* @__PURE__ */ new Set()).delete(_))), _.f & 33554432) {
			if (_.f ^= ne, _ === l) Rr(_, null, n);
			else {
				var y = d ? d.next : l;
				_ === e.effect.last && (e.effect.last = _.prev), _.prev && (_.prev.next = _.next), _.next && (_.next.prev = _.prev), zr(e, d, _), zr(e, _, y), Rr(_, y, n), d = _, p = [], m = [], l = Fr(d.next);
				continue;
			}
		}
		if (_ !== l) {
			if (u !== void 0 && u.has(_)) {
				if (p.length < m.length) {
					var b = m[0], x;
					d = b.prev;
					var S = p[0], ee = p[p.length - 1];
					for (x = 0; x < p.length; x += 1) Rr(p[x], b, n);
					for (x = 0; x < m.length; x += 1) u.delete(m[x]);
					zr(e, S.prev, ee.next), zr(e, d, S), zr(e, ee, b), l = b, d = ee, --v, p = [], m = [];
				} else u.delete(_), Rr(_, l, n), zr(e, _.prev, _.next), zr(e, _, d === null ? e.effect.first : d.next), zr(e, d, _), d = _;
				continue;
			}
			for (p = [], m = []; l !== null && l !== _;) (u ??= /* @__PURE__ */ new Set()).add(l), m.push(l), l = Fr(l.next);
			if (l === null) continue;
		}
		_.f & 33554432 || p.push(_), d = _, l = Fr(_.next);
	}
	if (e.outrogroups !== null) {
		for (let t of e.outrogroups) t.pending.size === 0 && (Mr(e, r(t.done)), e.outrogroups?.delete(t));
		e.outrogroups.size === 0 && (e.outrogroups = null);
	}
	if (l !== null || u !== void 0) {
		var C = [];
		if (u !== void 0) for (_ of u) _.f & 8192 || C.push(_);
		for (; l !== null;) !(l.f & 8192) && l !== e.fallback && C.push(l), l = Fr(l.next);
		var te = C.length;
		if (te > 0) {
			var re = i & 4 && s === 0 ? n : null;
			if (o) {
				for (v = 0; v < te; v += 1) C[v].nodes?.a?.measure();
				for (v = 0; v < te; v += 1) C[v].nodes?.a?.fix();
			}
			jr(e, C, re);
		}
	}
	o && Je(() => {
		if (f !== void 0) for (_ of f) _.nodes?.a?.apply();
	});
}
function Lr(e, t, n, r, i, a, o, s) {
	var c = o & 1 ? o & 16 ? Rt(n) : /* @__PURE__ */ N(n, !1, !1) : null, l = o & 2 ? Rt(i) : null;
	return {
		v: c,
		i: l,
		e: V(() => (a(t, c ?? n, l ?? i, s), () => {
			e.delete(r);
		}))
	};
}
function Rr(e, t, n) {
	if (e.nodes) for (var r = e.nodes.start, i = e.nodes.end, a = t && !(t.f & 33554432) ? t.nodes.start : n; r !== null;) {
		var o = /* @__PURE__ */ tn(r);
		if (a.before(r), r === i) return;
		r = o;
	}
}
function zr(e, t, n) {
	t === null ? e.effect.first = n : t.next = n, n === null ? e.effect.last = t : n.prev = t;
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/blocks/svelte-head.js
function Br(e, t) {
	let n = null, r = E;
	var i;
	if (E) {
		n = D;
		for (var a = /* @__PURE__ */ en(document.head); a !== null && (a.nodeType !== 8 || a.data !== e);) a = /* @__PURE__ */ tn(a);
		if (a === null) be(!1);
		else {
			var o = /* @__PURE__ */ tn(a);
			a.remove(), O(o);
		}
	}
	E || (i = document.head.appendChild(F()));
	try {
		xn(() => {
			var e = V(() => t(i));
			e.f |= ee, E || (e.nodes === null ? e.nodes = {
				start: i,
				end: i,
				a: null,
				t: null
			} : e.nodes.end = i);
		});
	} finally {
		r && (be(!0), O(n));
	}
}
//#endregion
//#region node_modules/svelte/src/internal/shared/attributes.js
var Vr = [..." 	\n\r\f\xA0\v﻿"];
function Hr(e, t, n) {
	var r = e == null ? "" : "" + e;
	if (t && (r = r ? r + " " + t : t), n) {
		for (var i of Object.keys(n)) if (n[i]) r = r ? r + " " + i : i;
		else if (r.length) for (var a = i.length, o = 0; (o = r.indexOf(i, o)) >= 0;) {
			var s = o + a;
			(o === 0 || Vr.includes(r[o - 1])) && (s === r.length || Vr.includes(r[s])) ? r = (o === 0 ? "" : r.substring(0, o)) + r.substring(s + 1) : o = s;
		}
	}
	return r === "" ? null : r;
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/elements/class.js
function Ur(e, t, n, r, i, a) {
	var o = e[le];
	if (E || o !== n || o === void 0) {
		var s = Hr(n, r, a);
		(!E || s !== e.getAttribute("class")) && (s == null ? e.removeAttribute("class") : t ? e.className = s : e.setAttribute("class", s)), e[le] = n;
	} else if (a && i !== a) for (var c in a) {
		var l = !!a[c];
		(i == null || l !== !!i[c]) && e.classList.toggle(c, l);
	}
	return a;
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/elements/attributes.js
var Wr = Symbol("is custom element"), Gr = Symbol("is html"), Kr = me ? "link" : "LINK";
function qr(e) {
	if (E) {
		var t = !1, n = () => {
			if (!t) {
				if (t = !0, e.hasAttribute("value")) {
					var n = e.value;
					Jr(e, "value", null), e.value = n;
				}
				if (e.hasAttribute("checked")) {
					var r = e.checked;
					Jr(e, "checked", null), e.checked = r;
				}
			}
		};
		e[fe] = n, Je(n), et();
	}
}
function Jr(e, t, n, r) {
	var i = Yr(e);
	E && (i[t] = e.getAttribute(t), t === "src" || t === "srcset" || t === "href" && e.nodeName === Kr) || i[t] !== (i[t] = n) && (t === "loading" && (e[se] = n), n == null ? e.removeAttribute(t) : typeof n != "string" && Zr(e).has(t) ? e[t] = n : e.setAttribute(t, n));
}
function Yr(e) {
	return e[ce] ??= {
		[Wr]: e.nodeName.includes("-"),
		[Gr]: e.namespaceURI === ge
	};
}
var Xr = /* @__PURE__ */ new Map();
function Zr(e) {
	var t = e.getAttribute("is") || e.nodeName, n = Xr.get(t);
	if (n) return n;
	Xr.set(t, n = /* @__PURE__ */ new Set());
	for (var r, i = e, a = Element.prototype; a !== i;) {
		for (var s in r = o(i), r) r[s].set && s !== "innerHTML" && s !== "textContent" && s !== "innerText" && n.add(s);
		i = l(i);
	}
	return n;
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/elements/bindings/input.js
function Qr(e, t, n = t) {
	var r = /* @__PURE__ */ new WeakSet();
	nt(e, "input", async (i) => {
		var a = i ? e.defaultValue : e.value;
		if (a = $r(e) ? ei(a) : a, n(a), M !== null && r.add(M), await $n(), a !== (a = t())) {
			var o = e.selectionStart, s = e.selectionEnd, c = e.value.length;
			if (e.value = a ?? "", s !== null) {
				var l = e.value.length;
				o === s && s === c && l > c ? (e.selectionStart = l, e.selectionEnd = l) : (e.selectionStart = o, e.selectionEnd = Math.min(s, l));
			}
		}
	}), (E && e.defaultValue !== e.value || nr(t) == null && e.value) && (n($r(e) ? ei(e.value) : e.value), M !== null && r.add(M)), bn(() => {
		var n = t();
		if (e === document.activeElement) {
			var i = M;
			if (r.has(i)) return;
		}
		$r(e) && n === ei(e.value) || (e.type !== "date" || n || e.value) && n !== e.value && (e.value = n ?? "");
	});
}
function $r(e) {
	var t = e.type;
	return t === "number" || t === "range";
}
function ei(e) {
	return e === "" ? null : +e;
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/elements/bindings/this.js
function ti(e, t) {
	return e === t || e?.[ae] === t;
}
function ni(e = We(), t, n, r) {
	var i = A.r, a = G;
	return vn(() => {
		var o, s;
		return bn(() => {
			o = s, s = r?.() || [], nr(() => {
				ti(n(...s), e) || (t(e, ...s), o && ti(n(...o), e) && t(null, ...o));
			});
		}), () => {
			let r = a;
			for (; r !== i && r.parent !== null && r.parent.f & 33554432;) r = r.parent;
			let o = () => {
				s && ti(n(...s), e) && t(null, ...s);
			}, c = r.teardown;
			r.teardown = () => {
				o(), c?.();
			};
		};
	}), e;
}
//#endregion
//#region node_modules/svelte/src/internal/client/dom/legacy/lifecycle.js
function ri(e = !1) {
	let t = A, n = t.l.u;
	if (!n) return;
	let r = () => rr(t.s);
	if (e) {
		let e = 0, n = {}, i = /* @__PURE__ */ st(() => {
			let r = !1, i = t.s;
			for (let e in i) i[e] !== n[e] && (n[e] = i[e], r = !0);
			return r && e++, e;
		});
		r = () => Y(i);
	}
	n.b.length && gn(() => {
		ii(t, r), p(n.b);
	}), mn(() => {
		let e = nr(() => n.m.map(f));
		return () => {
			for (let t of e) typeof t == "function" && t();
		};
	}), n.a.length && mn(() => {
		ii(t, r), p(n.a);
	});
}
function ii(e, t) {
	if (e.l.s) for (let t of e.l.s) Y(t);
	t();
}
function ai(e) {
	A === null && Oe("onMount"), ze && A.l !== null ? oi(A).m.push(e) : mn(() => {
		let t = nr(e);
		if (typeof t == "function") return t;
	});
}
function oi(e) {
	var t = e.l;
	return t.u ??= {
		a: [],
		b: [],
		m: []
	};
}
//#endregion
//#region node_modules/svelte/src/internal/flags/legacy.js
typeof window < "u" && ((window.__svelte ??= {}).v ??= /* @__PURE__ */ new Set()).add("5"), Be();
//#endregion
//#region src/App.svelte
var si = /* @__PURE__ */ Z("<span class=\"identity\"> </span>"), ci = /* @__PURE__ */ Z("<span></span>"), li = /* @__PURE__ */ Z("<div class=\"camera-wrap\"><video playsinline=\"\"></video><div class=\"face-oval\"></div><div class=\"scan-line\"></div></div> <div class=\"copy\"><p class=\"eyebrow\">IDENTITY CHECK</p><h1>Look at the camera</h1><p> </p></div> <button class=\"secondary\">Enroll yourself</button>", 3), ui = /* @__PURE__ */ Z("<div class=\"symbol success\">✓</div> <div class=\"copy\"><p class=\"eyebrow\">VERIFIED</p><h1> </h1><p>Your identity has been confirmed.</p></div> <button class=\"primary\">Continue <span>→</span></button>", 1), di = /* @__PURE__ */ Z("<em><small>Taken by</small> </em>"), fi = /* @__PURE__ */ Z("<span class=\"gun-cell\"><span></span> <b> </b> <!></span>"), pi = /* @__PURE__ */ Z("<span><i class=\"gun-fault\"></i>No data yet</span>"), mi = /* @__PURE__ */ Z("<span><i class=\"gun-in\"></i>Present</span><span><i class=\"gun-out\"></i>Missing</span><span><i class=\"gun-detecting\"></i>Sensor not there</span><!>", 1), hi = /* @__PURE__ */ Z("<span><i class=\"gun-detecting\"></i>No sensor data is being received</span>"), gi = /* @__PURE__ */ Z("<button class=\"choice-card locker-choice\"><span class=\"locker-head\"><span class=\"choice-copy\"><strong> </strong><small> </small></span> <span><i></i> </span> <span> </span> <span class=\"chevron\">›</span></span> <span></span> <span class=\"gun-legend\"><!></span></button>"), _i = /* @__PURE__ */ Z("<div class=\"copy\"><p class=\"eyebrow\">STEP 1 OF 3</p><h1>Choose a locker</h1><p>Select the inventory you want to request from.</p></div> <div class=\"cards\"></div> <button class=\"quiet\">Cancel</button>", 1), vi = /* @__PURE__ */ Z("<button><span class=\"tick\">✓</span> <span></span> <b> </b> <small> </small></button>"), yi = /* @__PURE__ */ Z("<button class=\"quiet\">Select all available</button>"), bi = /* @__PURE__ */ Z("<button class=\"back\">‹ Back</button> <div class=\"copy\"><p class=\"eyebrow\">STEP 2 OF 3</p><h1> </h1><p> </p></div> <div></div> <!> <button class=\"primary\"> <span>→</span></button>", 1), xi = /* @__PURE__ */ Z("<button class=\"back\">‹ Back</button> <div class=\"copy\"><p class=\"eyebrow\">STEP 3 OF 3</p><h1>Review your request</h1><p>Nothing opens until an admin approves it.</p></div> <div class=\"summary\"><div><span>Locker</span><strong> </strong></div><div><span>Type</span><strong class=\"capitalize\"> </strong></div><div><span> </span><strong> </strong></div></div> <label class=\"field\"><span>Reason <i>Optional</i></span><input maxlength=\"120\" placeholder=\"e.g. Range practice\"/></label> <button class=\"primary\"> </button>", 1), Si = /* @__PURE__ */ Z("<div class=\"progress-ring\"><span></span></div> <div class=\"copy\"><p class=\"eyebrow\">REQUEST SENT</p><h1>Waiting for approval</h1><p>An admin has been notified. Keep this screen open.</p></div> <div class=\"request-pill\"><span> </span><strong> </strong></div> <button class=\"quiet danger\">Cancel request</button>", 1), Ci = /* @__PURE__ */ Z("<div><span> </span></div>"), wi = /* @__PURE__ */ Z("<div> </div> <div class=\"copy\"><p class=\"eyebrow\"> </p><h1> </h1><p> </p></div> <div class=\"slots\"></div>", 1), Ti = /* @__PURE__ */ Z("<div class=\"symbol success\">✓</div><div class=\"copy\"><p class=\"eyebrow\">COLLECTED</p><h1> </h1><p> </p></div> <button class=\"primary\">Done</button>", 1), Ei = /* @__PURE__ */ Z("<div class=\"symbol danger-symbol\">×</div><div class=\"copy\"><p class=\"eyebrow\">NOT APPROVED</p><h1>Request declined</h1><p>Ask an administrator if you need help.</p></div> <button class=\"primary\">Start over</button>", 1), Di = /* @__PURE__ */ Z("<div class=\"symbol success\">✓</div><div class=\"copy\"><p class=\"eyebrow\">COMPLETE</p><h1>Returned safely</h1><p>The slot is locked again. Thank you.</p></div> <button class=\"primary\">Done</button>", 1), Oi = /* @__PURE__ */ Z("<button class=\"back\">‹ Cancel</button> <div class=\"copy\"><p class=\"eyebrow\">ENROLLMENT · STEP 1 OF 2</p><h1>Tell us who you are</h1><p>An admin will verify these details before activating your face.</p></div> <div class=\"form-stack\"><label class=\"field\"><span>Full name</span><input autocomplete=\"name\" placeholder=\"Your full name\"/></label><label class=\"field\"><span>Service number</span><input autocomplete=\"off\" placeholder=\"Your service number\"/></label></div> <button class=\"primary\">Continue <span>→</span></button>", 1), ki = /* @__PURE__ */ Z("<button class=\"back\">‹ Back</button> <div class=\"camera-wrap compact\"><video playsinline=\"\"></video><div class=\"face-oval\"></div></div> <div class=\"copy\"><p class=\"eyebrow\">ENROLLMENT · STEP 2 OF 2</p><h1>Capture your face</h1><p> </p></div> <button class=\"primary\"> </button>", 3), Ai = /* @__PURE__ */ Z("<div class=\"symbol success\">✓</div><div class=\"copy\"><p class=\"eyebrow\">SUBMITTED</p><h1>Request sent</h1><p>An admin must approve your enrollment before your face can be used.</p></div> <button class=\"primary\">Return to face scan</button>", 1), ji = /* @__PURE__ */ Z("<p class=\"error\" role=\"alert\"> </p>"), Mi = /* @__PURE__ */ Z("<section><!> <!></section>"), Ni = /* @__PURE__ */ Z("<div class=\"shell\"><header class=\"topbar\"><div class=\"brand\"><span class=\"brand-mark\">A</span><span>Armory</span></div> <span class=\"context\"> </span> <!></header> <main><!></main> <footer><span>Secure local system</span><span class=\"step-dot\"></span><span> </span></footer></div>");
function Pi(e, t) {
	He(t, !1);
	let n = /* @__PURE__ */ N(new URLSearchParams(location.search).has("enroll") ? "enroll-name" : "face"), r = /* @__PURE__ */ N(1), i = /* @__PURE__ */ N(null), a = /* @__PURE__ */ N(), o = /* @__PURE__ */ N("Preparing the camera…"), s, c, l = /* @__PURE__ */ N([]), u = /* @__PURE__ */ N(null), d = /* @__PURE__ */ N([]), f = /* @__PURE__ */ N(""), p = /* @__PURE__ */ N(null), m = /* @__PURE__ */ N(""), h = /* @__PURE__ */ N(!1), g = /* @__PURE__ */ N({
		name: "",
		serviceNo: ""
	}), _ = {
		face: "Face verification",
		welcome: "Welcome",
		catalog: "Choose a locker",
		guns: "Choose guns",
		review: "Review",
		waiting: "Approval",
		open: "Locker access",
		correct: "Collected",
		wrong: "Wrong slot",
		declined: "Declined",
		returned: "Complete",
		"enroll-name": "Your details",
		"enroll-face": "Face capture",
		"enroll-pending": "Request sent"
	};
	function v(e, t = !0) {
		P(r, t ? 1 : -1), P(m, ""), P(n, e);
	}
	async function y(e, t, n) {
		let r = await fetch(t, {
			method: e,
			headers: n ? { "Content-Type": "application/json" } : {},
			body: n ? JSON.stringify(n) : void 0,
			credentials: "same-origin"
		}), i = {};
		try {
			i = await r.json();
		} catch {}
		if (!r.ok) throw Object.assign(Error(i.error || (typeof i == "string" ? i : "Something went wrong.")), { status: r.status });
		return i;
	}
	function b() {
		s && s(), s = null, Y(a) && window.armoryFace && window.armoryFace.stop(Y(a));
	}
	async function x() {
		clearInterval(c), b(), await fetch("/api/logout", {
			method: "POST",
			credentials: "same-origin"
		}).catch(() => {}), P(i, null), P(u, null), P(d, []), P(p, null), P(f, ""), v("face", !1), await $n(), P(o, "Preparing the camera…");
		try {
			await window.armoryFace.load(), await window.armoryFace.start(Y(a)), P(o, "Hold still and look straight ahead."), s = window.armoryFace.watch(Y(a), {
				onLooking: () => P(o, "Looking for your face…"),
				onNoMatch: () => P(o, "We don't recognise you yet."),
				onError: () => P(o, "Keep your face inside the frame."),
				onMatch: async (e) => {
					b(), P(i, e), v("welcome");
					try {
						le((await y("GET", "/api/requests/current")).id);
					} catch (e) {
						e.status !== 404 && P(m, e.message);
					}
				}
			});
		} catch (e) {
			P(o, e.message || "The camera is unavailable.");
		}
	}
	async function S() {
		P(h, !0);
		try {
			P(l, await y("GET", "/api/catalog")), v("catalog");
		} catch (e) {
			P(m, e.message);
		} finally {
			P(h, !1);
		}
	}
	async function ee() {
		try {
			P(l, await y("GET", "/api/catalog"));
		} catch {
			return;
		}
		if (!Y(u)) return;
		let e = Y(l).find((e) => e.id === Y(u).id);
		e && (P(u, e), P(d, Y(d).filter((t) => e.slots.find((e) => e.no === t)?.available)));
	}
	let C = () => Y(n) === "catalog" || Y(n) === "guns" || Y(n) === "review";
	function te(e) {
		P(u, e), P(d, []), v("guns");
	}
	function ne(e) {
		P(d, Y(d).includes(e) ? Y(d).filter((t) => t !== e) : [...Y(d), e].sort((e, t) => e - t));
	}
	function re(e, t) {
		return t.available ? "Available" : t.taken_by ? `Taken by ${t.taken_by}` : !e.online || t.reading === 2 ? "Sensor not there" : t.reading === 0 ? "Missing" : "No data yet";
	}
	let ie = (e) => e.length < 2 ? `${e[0] ?? ""}` : `${e.slice(0, -1).join(", ")} and ${e[e.length - 1]}`, w = (e, t) => (e?.chosen || []).filter((e) => !t || e.status === t).map((e) => e.no);
	function ae() {
		if (!window.EventSource) return () => {};
		let e, t = new EventSource("/events");
		return t.onmessage = () => {
			clearTimeout(e), e = setTimeout(() => {
				C() && ee();
			}, 150);
		}, () => {
			clearTimeout(e), t.close();
		};
	}
	function oe() {
		!document.hidden && C() && ee();
	}
	async function se() {
		P(h, !0), P(m, "");
		try {
			P(p, await y("POST", "/api/requests", {
				locker_id: Y(u).id,
				slots: Y(d),
				reason: Y(f)
			})), le(Y(p).id);
		} catch (e) {
			P(m, e.message);
		} finally {
			P(h, !1);
		}
	}
	function ce(e) {
		P(p, e), e.status === "pending" ? v("waiting") : e.status === "approved" ? v(e.wrong?.length ? "wrong" : "open") : e.status === "collected" ? v("correct") : e.status === "returned" ? (clearInterval(c), v("returned")) : e.status === "rejected" && (clearInterval(c), v("declined"));
	}
	function le(e) {
		clearInterval(c);
		let t = async () => {
			try {
				ce(await y("GET", `/api/requests/${e}`));
			} catch {}
		};
		t(), c = setInterval(t, 1e3);
	}
	async function ue() {
		Y(p) && await y("POST", `/api/requests/${Y(p).id}/cancel`).catch(() => {}), x();
	}
	async function de() {
		b(), v("enroll-name");
	}
	async function fe() {
		v("enroll-face"), await $n(), P(o, "Preparing the camera…");
		try {
			await window.armoryFace.load(), await window.armoryFace.start(Y(a)), P(o, "Center your face and hold still.");
		} catch (e) {
			P(o, e.message || "The camera is unavailable.");
		}
	}
	async function pe() {
		P(h, !0), P(m, "");
		try {
			let e = [];
			for (let t = 0; t < 3; t++) {
				P(o, `Capturing ${t + 1} of 3…`);
				let n = await window.armoryFace.descriptor(Y(a));
				n && e.push(Array.from(n)), await window.armoryFace.sleep(180);
			}
			if (e.length < 2) throw Error("We couldn't see your face clearly. Try again in better light.");
			await window.armoryFace.post("/enroll", {
				name: Y(g).name,
				service_no: Y(g).serviceNo,
				descriptors: e
			}), b(), v("enroll-pending");
		} catch (e) {
			P(m, e.message || "Could not send the enrollment request.");
		} finally {
			P(h, !1);
		}
	}
	ai(() => {
		Y(n) === "face" && x();
		let e = ae(), t = setInterval(oe, 1e3);
		return window.addEventListener("pageshow", oe), document.addEventListener("visibilitychange", oe), () => {
			b(), clearInterval(c), clearInterval(t), window.removeEventListener("pageshow", oe), document.removeEventListener("visibilitychange", oe), e();
		};
	}), ri();
	var me = Ni();
	Br("1n46o8q", (e) => {
		vn(() => {
			Yt.title = "Armory";
		});
	});
	var he = I(me), T = z(I(he), 2), ge = R(T, !0), _e = z(T, 2), ve = (e) => {
		var t = si(), n = R(t, !0);
		B(() => $(n, Y(i).name)), Q(e, t);
	}, ye = (e) => {
		Q(e, ci());
	};
	Dr(_e, (e) => {
		Y(i) ? e(ve) : e(ye, -1);
	}), k(he);
	var E = z(he, 2);
	kr(I(E), () => Y(n), (e) => {
		var t = Mi();
		let r;
		var s = I(t), c = (e) => {
			var t = li(), n = L(t), r = I(n);
			r.muted = !0, ni(r, (e) => P(a, e), () => Y(a)), Se(2), k(n);
			var i = z(n, 2), s = R(z(I(i), 2), !0);
			k(i);
			var c = z(i, 2);
			B(() => $(s, Y(o))), X("click", c, de), Q(e, t);
		}, _ = (e) => {
			var t = ui(), n = z(L(t), 2), r = R(z(I(n)));
			Se(), k(n);
			var a = z(n, 2);
			B((e, t) => {
				$(r, `Good ${e ?? ""}, ${t ?? ""}`), a.disabled = Y(h);
			}, [() => (/* @__PURE__ */ new Date()).getHours() < 12 ? "morning" : (/* @__PURE__ */ new Date()).getHours() < 18 ? "afternoon" : "evening", () => Y(i)?.name?.split(" ")[0]]), X("click", a, S), Q(e, t);
		}, y = (e) => {
			var t = _i(), n = z(L(t), 2);
			Pr(n, 5, () => Y(l), Ar, (e, t) => {
				var n = gi(), r = I(n), i = I(r), a = I(i), o = R(a, !0), s = R(z(a), !0);
				k(i);
				var c = z(i, 2);
				let l;
				var u = z(I(c), 1, !0);
				k(c);
				var d = z(c, 2);
				let f;
				var p = R(d, !0);
				Se(2), k(r);
				var m = z(r, 2);
				Pr(m, 5, () => Y(t).slots || [], Ar, (e, n) => {
					var r = fi(), i = I(r), a = z(i, 2), o = R(a, !0), s = z(a, 2), c = (e) => {
						var t = di(), r = z(I(t), 1, !0);
						k(t), B(() => $(r, Y(n).taken_by)), Q(e, t);
					};
					Dr(s, (e) => {
						Y(n).taken_by && e(c);
					}), k(r), B(() => {
						Ur(i, 1, `gun ${!Y(t).online || Y(n).reading === 2 ? "gun-detecting" : Y(n).reading === 1 ? "gun-in" : Y(n).reading === 0 ? "gun-out" : "gun-fault"}`), $(o, Y(n).no);
					}), Q(e, r);
				}), k(m);
				var h = z(m, 2), g = I(h), _ = (e) => {
					var n = mi(), r = z(L(n), 3), i = (e) => {
						Q(e, pi());
					}, a = /* @__PURE__ */ ut(() => Y(t).slots?.some((e) => e.reading < 0));
					Dr(r, (e) => {
						Y(a) && e(i);
					}), Q(e, n);
				}, v = (e) => {
					Q(e, hi());
				};
				Dr(g, (e) => {
					Y(t).online ? e(_) : e(v, -1);
				}), k(h), k(n), B(() => {
					n.disabled = Y(t).available === 0, $(o, Y(t).name), $(s, Y(t).location || Y(t).kind), l = Ur(c, 1, "board-state", null, l, { online: Y(t).online }), $(u, Y(t).online ? "Detecting" : "Not detecting"), f = Ur(d, 1, "count", null, f, { available: Y(t).available > 0 }), $(p, Y(t).available ? `${Y(t).available} available` : "Guns not available"), Ur(m, 1, `gun-row gun-row-${Y(t).kind ?? ""}`);
				}), X("click", n, () => te(Y(t))), Q(e, n);
			}), k(n), X("click", z(n, 2), x), Q(e, t);
		}, ee = (e) => {
			var t = bi(), n = L(t), r = z(n, 2), i = z(I(r)), a = R(i), o = R(z(i));
			k(r);
			var s = z(r, 2);
			let c;
			Pr(s, 5, () => Y(u)?.slots || [], Ar, (e, t) => {
				var n = vi();
				let r;
				var i = z(I(n), 2), a = z(i, 2), o = R(a), s = R(z(a, 2), !0);
				k(n), B((e, a, c) => {
					r = Ur(n, 1, "gun-pick", null, r, { picked: e }), Jr(n, "aria-pressed", a), n.disabled = !Y(t).available, Ur(i, 1, `gun ${!Y(u).online || Y(t).reading === 2 ? "gun-detecting" : Y(t).reading === 1 ? "gun-in" : Y(t).reading === 0 ? "gun-out" : "gun-fault"}`), $(o, `Gun ${Y(t).no ?? ""}`), $(s, c);
				}, [
					() => Y(d).includes(Y(t).no),
					() => Y(d).includes(Y(t).no),
					() => re(Y(u), Y(t))
				]), X("click", n, () => ne(Y(t).no)), Q(e, n);
			}), k(s);
			var l = z(s, 2), f = (e) => {
				var t = yi();
				X("click", t, () => P(d, Y(u).slots.filter((e) => e.available).map((e) => e.no))), Q(e, t);
			}, p = /* @__PURE__ */ ut(() => (Y(u)?.slots || []).filter((e) => e.available).length > 1);
			Dr(l, (e) => {
				Y(p) && e(f);
			});
			var m = z(l, 2), h = I(m);
			Se(), k(m), B(() => {
				$(a, `Choose your ${Y(u)?.kind ?? ""}s`), $(o, `Tap each gun you need from ${Y(u)?.name ?? ""}. You can pick more than one.`), c = Ur(s, 1, "gun-pick-row", null, c, { pistol: Y(u)?.kind === "pistol" }), m.disabled = !Y(d).length, $(h, `${Y(d).length > 1 ? `Continue with ${Y(d).length} guns` : Y(d).length ? "Continue with 1 gun" : "Pick at least one gun"} `);
			}), X("click", n, () => v("catalog", !1)), X("click", m, () => v("review")), Q(e, t);
		}, C = (e) => {
			var t = xi(), n = L(t), r = z(n, 4), i = I(r), a = R(z(I(i)), !0);
			k(i);
			var o = z(i), s = R(z(I(o)), !0);
			k(o);
			var c = z(o), l = I(c), p = R(l, !0), m = R(z(l), !0);
			k(c), k(r);
			var g = z(r, 2), _ = z(I(g));
			qr(_), k(g);
			var y = z(g, 2), b = R(y, !0);
			B((e) => {
				$(a, Y(u)?.name), $(s, Y(u)?.kind), $(p, Y(d).length > 1 ? "Guns" : "Gun"), $(m, e), y.disabled = Y(h), $(b, Y(h) ? "Sending…" : "Send request");
			}, [() => Y(d).join(", ")]), X("click", n, () => v("guns", !1)), Qr(_, () => Y(f), (e) => P(f, e)), X("click", y, se), Q(e, t);
		}, ae = (e) => {
			var t = Si(), n = z(L(t), 4), r = I(n), i = R(r, !0), a = R(z(r), !0);
			k(n);
			var o = z(n, 2);
			B((e) => {
				$(i, Y(u)?.name || Y(p)?.locker_name), $(a, e);
			}, [() => w(Y(p)).length ? `${w(Y(p)).length > 1 ? "Guns" : "Gun"} ${w(Y(p)).join(", ")}` : Y(u)?.kind || Y(p)?.kind]), X("click", o, ue), Q(e, t);
		}, oe = (e) => {
			var t = wi(), r = L(t);
			let i;
			var a = R(r, !0), o = z(r, 2), s = I(o), c = R(s, !0), l = z(s), u = R(l, !0), d = R(z(l), !0);
			k(o);
			var f = z(o, 2);
			Pr(f, 5, () => Y(p)?.slots || [], Ar, (e, t) => {
				var n = Ci();
				let r;
				var i = R(I(n), !0);
				k(n), B((e, a, o) => {
					r = Ur(n, 1, "", null, r, {
						target: e,
						done: a,
						wrong: o
					}), $(i, Y(t).no);
				}, [
					() => w(Y(p), "chosen").includes(Y(t).no),
					() => w(Y(p), "collected").includes(Y(t).no),
					() => Y(p)?.wrong?.includes(Y(t).no)
				]), Q(e, n);
			}), k(f), B((e) => {
				i = Ur(r, 1, "symbol", null, i, { "danger-symbol": Y(n) === "wrong" }), $(a, Y(n) === "wrong" ? "!" : "↗"), $(c, Y(n) === "wrong" ? "WRONG SLOT" : "APPROVED"), $(u, Y(n) === "wrong" ? "Put it back" : `${Y(p)?.locker_name} is opening`), $(d, e);
			}, [() => Y(n) === "wrong" ? `Return gun ${Y(p)?.wrong?.join(", ")}, then use the highlighted ${w(Y(p), "chosen").length > 1 ? "slots" : "slot"}.` : w(Y(p), "collected").length ? `Now take gun ${ie(w(Y(p), "chosen"))}.` : `Take ${w(Y(p)).length > 1 ? `guns ${ie(w(Y(p)))}` : `the ${Y(p)?.kind} from slot ${Y(p)?.slot_no}`}.`]), Q(e, t);
		}, ce = (e) => {
			var t = Ti(), n = z(L(t)), r = z(I(n)), i = R(r, !0), a = R(z(r), !0);
			k(n);
			var o = z(n, 2);
			B((e, t) => {
				$(i, e), $(a, t);
			}, [() => w(Y(p)).length > 1 ? "All guns collected" : "Correct item", () => w(Y(p)).length > 1 ? `Return them to ${Y(p)?.locker_name}, slots ${ie(w(Y(p)))}, when you are finished.` : `Return it to ${Y(p)?.locker_name}, slot ${Y(p)?.slot_no}, when you are finished.`]), X("click", o, x), Q(e, t);
		}, le = (e) => {
			var t = Ei();
			X("click", z(L(t), 3), x), Q(e, t);
		}, me = (e) => {
			var t = Di();
			X("click", z(L(t), 3), x), Q(e, t);
		}, he = (e) => {
			var t = Oi(), n = L(t), r = z(n, 4), i = I(r), a = z(I(i));
			qr(a), k(i);
			var o = z(i), s = z(I(o));
			qr(s), k(o), k(r);
			var c = z(r, 2);
			B((e) => c.disabled = e, [() => !Y(g).name.trim() || !Y(g).serviceNo.trim()]), X("click", n, x), Qr(a, () => Y(g).name, (e) => Bt(g, Y(g).name = e)), Qr(s, () => Y(g).serviceNo, (e) => Bt(g, Y(g).serviceNo = e)), X("click", c, fe), Q(e, t);
		}, T = (e) => {
			var t = ki(), n = L(t), r = z(n, 2), i = I(r);
			i.muted = !0, ni(i, (e) => P(a, e), () => Y(a)), Se(), k(r);
			var s = z(r, 2), c = R(z(I(s), 2), !0);
			k(s);
			var l = z(s, 2), u = R(l, !0);
			B(() => {
				$(c, Y(o)), l.disabled = Y(h), $(u, Y(h) ? "Capturing…" : "Send enrollment request");
			}), X("click", n, () => {
				b(), v("enroll-name", !1);
			}), X("click", l, pe), Q(e, t);
		}, ge = (e) => {
			var t = Ai();
			X("click", z(L(t), 3), x), Q(e, t);
		};
		Dr(s, (e) => {
			Y(n) === "face" ? e(c) : Y(n) === "welcome" ? e(_, 1) : Y(n) === "catalog" ? e(y, 2) : Y(n) === "guns" ? e(ee, 3) : Y(n) === "review" ? e(C, 4) : Y(n) === "waiting" ? e(ae, 5) : Y(n) === "open" || Y(n) === "wrong" ? e(oe, 6) : Y(n) === "correct" ? e(ce, 7) : Y(n) === "declined" ? e(le, 8) : Y(n) === "returned" ? e(me, 9) : Y(n) === "enroll-name" ? e(he, 10) : Y(n) === "enroll-face" ? e(T, 11) : Y(n) === "enroll-pending" && e(ge, 12);
		});
		var _e = z(s, 2), ve = (e) => {
			var t = ji(), n = R(t, !0);
			B(() => $(n, Y(m))), Q(e, t);
		};
		Dr(_e, (e) => {
			Y(m) && e(ve);
		}), k(t), B(() => r = Ur(t, 1, "screen", null, r, { "catalog-screen": Y(n) === "catalog" })), Q(e, t);
	}), k(E);
	var be = z(E, 2), D = R(z(I(be), 2), !0);
	k(be), k(me), B(() => {
		Jr(me, "data-direction", Y(r)), $(ge, _[Y(n)]), $(D, _[Y(n)]);
	}), Q(e, me), Ue();
}
//#endregion
//#region src/main.js
Sr(Pi, { target: document.getElementById("app") });
//#endregion
