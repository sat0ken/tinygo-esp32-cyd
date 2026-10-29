package train

// The levels of TRAIN (src/levels.cpp), a character a tile, a row of the
// board a line:
//
//	.  empty          a  crystal     h  giraffe     o  elephant
//	#  wall           b  crown       i  ice cream   p  drinking glass
//	>  locomotive     c  tree        j  cake        q  money
//	+  gate           d  apple       k  computer    r  airplane
//	                  e  cow         l  car         s  Lemmings
//	                  f  cherry      m  balloon
//	                  g  pool        n  alarm clock
//
// Level 0 is the demo scene.
const levels = "" +
	"...................." + // 0
	"aaaaaccc...b..jd...d" +
	"..a..c..c.b.b.jdd..d" +
	"..a..c..cb...bjd.d.d" +
	"..a..ccc.b...bjd.d.d" +
	"..a..c..cbbbbbjd.d.d" +
	"..a..c..cb...bjd..dd" +
	"..a..c..cb...bjd...d" +
	"...................." +
	">..................." +
	"............s......." +
	"#########+##########" +
	"####################" + // 1
	"#..................#" +
	"#...aaaaaaaaaaaa...#" +
	"#..................#" +
	"#..................#" +
	"#>....aaaaaaaa.....#" +
	"#..................#" +
	"#..................#" +
	"#.......aaaa.......#" +
	"#..................+" +
	"#..................#" +
	"####################" +
	"####################" + // 2
	"#..................#" +
	"#..................#" +
	"#.....a.a.a.a......#" +
	"#.....b.b.b.b......#" +
	"#.....bbbbbbb......#" +
	"#.....bbbbbbb......#" +
	"#.....bbbbbbb......#" +
	"#......#####.......#" +
	"#..>...............#" +
	"#..................#" +
	"#########+##########" +
	"####################" + // 3
	"#..................#" +
	"#..a.....#.........#" +
	"#.aaa....#...c.c...#" +
	"#.aba....#..c.s.c..#" +
	"#.aaa....#...c.c...#" +
	"#..a.....+..c.s.c..#" +
	"#........#...c.c...#" +
	"#........#.........#" +
	"#..>.....#.........#" +
	"#........#.........#" +
	"####################" +
	"#########+##########" + // 4
	"#.a.a.a.a.a.a.a.a..#" +
	"#..................#" +
	"#..................#" +
	"#..................#" +
	"#>.......d.........#" +
	"#.......ddd........#" +
	"#......ddsdd.......#" +
	"#..#..ddc#cdd..#...#" +
	"#..#.ddcd#dcdd.#...#" +
	"#..#ddcdddddcdd#...#" +
	"####################" +
	"####################" + // 5
	"#..................#" +
	"#.cccccccscccccccc.#" +
	"#..................#" +
	"#..e.e.e.#..e.e.e..#" +
	"#>..e.e..#...e.e...#" +
	"#..e.e.e.#..e.e.e..#" +
	"#...e.e..#...e.e...#" +
	"+..e.e.e.#..e.e.e..#" +
	"#........#.........#" +
	"#........b.........#" +
	"####################" +
	"####################" + // 6
	"#............s.....#" +
	"#.......fff..#.a...#" +
	"#......fffff.#.a...#" +
	"#......fffff.#.a...#" +
	"#>.....ffbff.#.a...#" +
	"#......fffff.#.a...#" +
	"#......fffff.#.a...#" +
	"#.......fff..#.a...#" +
	"##############.a...#" +
	"+..ccccccccc.......#" +
	"####################" +
	"#########+##########" + // 7
	"#..................#" +
	"#..................#" +
	"#...a...csc...e.e..#" +
	"#...#..c.g.c.......#" +
	"#>..a..cg.gc...e.e.#" +
	"#...#..c.g.c.......#" +
	"#...a..cg.gc..e.e..#" +
	"#...#..c.#.c.......#" +
	"#...a...c#c....e.e.#" +
	"#........#.........#" +
	"####################" +
	"####################" + // 8
	"#..................#" +
	"#..a.####.####.a...#" +
	"#..a.#.......#.a...#" +
	"#..a.#..c.c..#.a...#" +
	"#>.a.#.d.h.d.#.a...#" +
	"#..a.#..c.c..#.a...#" +
	"#..a.#...g...#.a...#" +
	"#..a.#.c...c.#.a...#" +
	"#..a.#########.a...#" +
	"#........+.........#" +
	"####################" +
	"####################" + // 9
	"#dddddddddddddddddd#" +
	"#........cc........#" +
	"#......######......#" +
	"#......#....#......#" +
	"#..aa..#.ii.#..aa..#" +
	"#.aaaa.#.ii.#.aaaa.#" +
	"#.aaaa.#.ii.#.aaaa.#" +
	"#..aa..#.ii.#..aa..#" +
	"#......##..##......#" +
	"#........>.........#" +
	"#########+##########" +
	"####################" + // 10
	"#..c...i.i.i...c...#" +
	"#..c....i.i....c...#" +
	"#..c.#...a...#.c...#" +
	"#..c.#...b...#.c...#" +
	"#>.g.#..#+#..#.h...#" +
	"#..c.#e.#j#.e#.c...#" +
	"#..c.#..#a#..#.c...#" +
	"#..c.#..#a#..#.c...#" +
	"#..c.#..#a#..#.c...#" +
	"#..s...........s...#" +
	"####################" +
	"####################" + // 11
	"#......g.##.g......#" +
	"#......#.kk.#......#" +
	"#..kk.########.kk..#" +
	"#..kk.#..bb..#.kk..#" +
	"#>.k...+....b...k..#" +
	"#..k.#c#k..k#c#.k..#" +
	"#..k.#c#.kk.#c#.k..#" +
	"#.kk.#c#.kk.#c#.kk.#" +
	"#.kk.#c#kkkk#c#.kk.#" +
	"#......#....#......#" +
	"####################" +
	"#########+##########" + // 12
	"#..................#" +
	"#.eeeeeeeeeeeeeeee.#" +
	"#.e..............e.#" +
	"#.e..............e.#" +
	"#>e..............e.#" +
	"#.e.j..........j.e.#" +
	"#.e.#####s######.e.#" +
	"#.e.#llll.lllll#.e.#" +
	"#.e.#llll.lllll#.e.#" +
	"#...#llll.lllll#...#" +
	"####################" +
	"####################" + // 13
	"##ii##...........dd#" +
	"##iii#...m.......dd#" +
	"##isi#..m.m......###" +
	"###ii#...m.hcb.....#" +
	"#>........ec##.a...#" +
	"#.......hc###......#" +
	"#.....ec###a.a....j#" +
	"#...hc###a.......###" +
	"#..c###a..a.a.a..dd#" +
	"#.e##...a...lll..dd#" +
	"#######+############" +
	"####################" + // 14
	"#nnn.n.n..n........#" +
	"#.n..n.n.n.........#" +
	"#.n..n.nn...nnn....#" +
	"#.n..n.n.n.........#" +
	"+>n..n.n..n........#" +
	"#..................#" +
	"#.......nnn..n..n.n#" +
	"#........n..n.n.nn.#" +
	"#........n..nnn.n.n#" +
	"#........n..n.n.n.n#" +
	"####################" +
	"####################" + // 15
	"#........a.........#" +
	"#...#####b######...#" +
	"#...#oco....gcg#...#" +
	"#...#coc.#f.cgc#...#" +
	"#>..+###kd#l####...#" +
	"#...####id#i####...#" +
	"#...#chcj#ijcec#...#" +
	"#...#hchjjjjece#...#" +
	"#...############...#" +
	"#..................#" +
	"####################" +
	"####################" + // 16
	"#..................#" +
	"#....#ppppppppp#...#" +
	"#.....#ppppppp#....#" +
	"#......#ppppp#.....#" +
	"#.......#ppp#......#" +
	"#..>.....#+#.......#" +
	"#........a#a.......#" +
	"#........a#a.......#" +
	"#........###.......#" +
	"#.......bbbbb......#" +
	"####################" +
	"####################" + // 17
	"#abablllll.........#" +
	"#baba####q#####....#" +
	"#ab#b#qqqq#qqq#....#" +
	"#baba#q####q#q#....#" +
	"#abab#qqqqqq#q#....#" +
	"#>a#########qq#....#" +
	"#abab#qqqq#qq##....#" +
	"#b##a#q##qqqqq#....#" +
	"#abab#qqq###qq#....#" +
	"#baba#qqqqqqqq#....#" +
	"#############+######" +
	"####################" + // 18
	"#....#.......+.....#" +
	"#.j..#r.#.#.r#...i.#" +
	"#.j..#..#.#..#..ii.#" +
	"#.jj.###...###.iii.#" +
	"#.jj..a..h..a.iiii.#" +
	"#>jj..a..o..a.iiii.#" +
	"#.jj.###...###.iii.#" +
	"#.j..#..#.#..#..ii.#" +
	"#.j..#r.#.#.r#...i.#" +
	"#....#.......#.....#" +
	"####################" +
	"####################" + // 19
	"#aaaaaaaaaaaaaaaaaa#" +
	"#aaaaaaaaaaaaaaaaaa#" +
	"#aaaaaaaaaaaaaaaaaa#" +
	"#aaaaaaaaaaaaaaaaaa#" +
	"#>aaaaaa+aaaaaaaaaa#" +
	"#aaaaaaaaaaaaaaaaaa#" +
	"#aaaaaaaaaaaaaaaaaa#" +
	"#aaaaaaaaaaaaaaaaaa#" +
	"#aaaaaaaaaaaaaaaaaa#" +
	"#aaaaaaaaaaaaaaaaaa#" +
	"####################" +
	"####################" + // 20
	"#..................#" +
	"#....b.....oooo....#" +
	"#....##....####....#" +
	"#...#a.#..#chch#...#" +
	"#>......##chchch#..#" +
	"#...###..chchchc#..#" +
	"#......#.hchchchc#a#" +
	"#.......##hchshch+.#" +
	"#.......#.#######..#" +
	"#fdfdfd##.....##.df#" +
	"####################" +
	"####################" + // 21
	"#cccc#........#ccsc#" +
	"#.k...#......#...k.#" +
	"#n.j...#....#...j.n#" +
	"####i...#..#...i####" +
	"#...#i...##...i#...#" +
	"#aa..#i..>...i#..aa#" +
	"#a#a..#i.pp.i#..a#a#" +
	"#..#a..#.pp.#..a#..#" +
	"#.b.#a..#pp#..a#.b.#" +
	"#....#a..pp..a#....+" +
	"####################" +
	"####################" + // 22
	"#>cccccc.c#c.cccccc#" +
	"#.ccsccccc#cccccscc#" +
	"#cccccc.c.#.c.ccccc#" +
	"#.cccc.c#####c.c.cc#" +
	"#cccc.#.#...#.#.ccc#" +
	"#.ccc.#.#.g.#.#.ccc#" +
	"#..c..#.#...#.#..c.#" +
	"#.cccc#.##.##.#cccc#" +
	"#cccc##.......##ccc#" +
	"+.c.c.#ccchccc#.c.c#" +
	"####################" +
	"####################" + // 23
	"#....pppp..pp......#" +
	"#...p####p.pp......#" +
	"#..p######ppp......#" +
	"#.p########pp......#" +
	"#>p##mmm####p......#" +
	"#.p##mrmiii#p.c.c.c#" +
	"#.p##mmmisi#pcccccc#" +
	"#.p#####iii#p.c.c.c#" +
	"#.p#####ii##p.c.c.c#" +
	"#.ppppppppppp.c.c.c+" +
	"####################" +
	"####################" + // 24
	"#.f..#.#...#.#...f.#" +
	"#.f..#.#...#.#...f.#" +
	"####.#.#.g.#.#.#####" +
	"#.f..#.#.h.#.#...f.#" +
	"#.####.#...#.#####.#" +
	"#.f....#...#.....f.#" +
	"#.####.##>##.#####.#" +
	"#.f..#..d.d..#...f.#" +
	"#....#c.....c#.....#" +
	"#.f..#ddddddd#...f.#" +
	"#########+##########" +
	"####################" + // 25
	"#>aaaaaaaaaaaaaaaaa#" +
	"#.b.b.b.b..b.b.b.b.#" +
	"##jjaaaaaaaaaaaajj##" +
	"##jj#jj.b..b.jj#jj##" +
	"##jj#jj#j#jj#jj#jj##" +
	"##jj#jj#j#jj#jj#jj##" +
	"##jj#jj#j#jj#jj#jj##" +
	"##jj#jj#j#jj#jj#jj##" +
	"##jj#jj#j#jj#jj#jj##" +
	"##jj#jj#j#jj#jj#jj##" +
	"########+###########" +
	"####################" + // 26
	"######......d.f.####" +
	"##hoh#...ccgpd#f.###" +
	"##oho...c##cgpd#f.##" +
	"##hoh..c#..#cgpd#f.#" +
	"##oho.c#....#cgpd#.#" +
	"####.c#......#cgpd.#" +
	"#...c#........#cgpd#" +
	"#..c#..........#cgp#" +
	"#.c#............#cg#" +
	"#c>#............#+s#" +
	"####################" +
	"####################" + // 27
	"#>a##.a..#..a......#" +
	"#...#.#.#a.#####.#.#" +
	"###.#.#........#...#" +
	"#..a..#.#.##.##.a..#" +
	"#.#####a#+#.a..#.#.#" +
	"#....#..#b.a.#a....#" +
	"####.#.##.#.a#.#aa.#" +
	"#....#.a.#.a.#.a..##" +
	"#.##.##...#..#a..#.#" +
	"#.a...#...#.#.#.#..#" +
	"####################" +
	"####################" + // 28
	"#........+.........#" +
	"#...##........##...#" +
	"#..#hh#......#hh#..#" +
	"#.#hhhh#.##.#hhhh#.#" +
	"#>#hhh...oo...hhh#.#" +
	"#.#hhh.hoooo#.hhh#.#" +
	"#.#hhhh######hhhh#.#" +
	"#..#hh#......#hh#..#" +
	"#...##........##...#" +
	"#..................#" +
	"####################" +
	"####################" + // 29
	"#....#....#...#....#" +
	"#..h.#.e..#.o.#.h..#" +
	"#....#....#...#....#" +
	"###.###.##>#.###.###" +
	"#c...c....c...c...c#" +
	"#s...c....c...c...s#" +
	"###.###.##+#.###.###" +
	"#....#...#....#....#" +
	"#..e.#.o.#..h.#.e..#" +
	"#....#...#....#....#" +
	"####################" +
	"####################" + // 30
	"#..................#" +
	"#..##############..#" +
	"#..#nnnnn##aaaaa#..#" +
	"#..#g#ggg##bbb#b#..#" +
	"#..#i##ii##dd##d#..#" +
	"#..#rl##l##k##kk#..#" +
	"#..#jjj#j##e#eee#..#" +
	"#..#mmmmm##ppppp#..#" +
	"#..####cc##cc####..#" +
	"#>.................+" +
	"####################" +
	"####################" + // 31
	"#.a.a.a.acca.a.a.a.#" +
	"#.################.#" +
	"#.a.a.a.acca.a.a.a.#" +
	"#.################.#" +
	"#>a.a.a.acca.a.a.a.#" +
	"#.################.#" +
	"#.a.a.a.acca.a.a.a.#" +
	"#.################.#" +
	"#.a.a.a.acca.a.a.a.#" +
	"#.#######+########.#" +
	"####################" +
	"####################" + // 32
	"#...#cccc##cccc#cc##" +
	"#..#cc#cc#cc#cc#ccc#" +
	"#c#cc#cc#cc#cc#cc#c#" +
	"#ccc#cc#cc#cc#cc#cc#" +
	"#>c##c##c##c##c##c##" +
	"#csc#cc#cs#cc#cs#cc#" +
	"#c#cc#cc#cc#cc#cc#c+" +
	"#cc#cc#cc#cc#cc#cc##" +
	"##cc#cc#cc#cc#cc#cc#" +
	"###cccc##cccc##cccc#" +
	"####################" +
	"####################" + // 33
	"#.i###i..jj..ffff..#" +
	"#.i###i.j##j.f###f.#" +
	"#..i#i.j####jf###f.#" +
	"#..i#i.j####jf###f.#" +
	"#>..i..j####jf###f.#" +
	"#...i..j####jffff.s+" +
	"#..i#i.j####jf#f...#" +
	"#..i#i.j####jf##f..#" +
	"#.i###i.j##j.f###f.#" +
	"#.i###i..jj..f###f.#" +
	"####################" +
	"####################" + // 34
	"#........>.........#" +
	"#...#pppppppppp#...#" +
	"#...##pppppppp##...#" +
	"#...###pppppp###...#" +
	"#...####pppp####...#" +
	"#...#####pp#####...#" +
	"#...####pppp####...#" +
	"#...###pppppp###...#" +
	"#...##pppppppp##...#" +
	"#...##pppppppp##...#" +
	"#########+##########" +
	"####################" + // 35
	"#........b.........#" +
	"#.......#+#........#" +
	"#......#e.h#.......#" +
	"#.....#ee.hh#......#" +
	"#>...#eee.hhh#.....#" +
	"#....#eee.hhh#.....#" +
	"#.....#ee.hh#......#" +
	"#......#e.h#.......#" +
	"#.......#.#........#" +
	"#cccsccccccccccsccc#" +
	"####################" +
	"####################" + // 36
	"##>#...#...#...#...#" +
	"##.#.b.#.b.#.b.#.b.#" +
	"##.#.#.#.#.#.#.#.#.#" +
	"##.#.#.#.#.#.#.#.#.#" +
	"##.#.#.#.#.#.#.#.#.#" +
	"##.#.#.#.#.#.#.#.#.#" +
	"##.#.#.#.#.#.#.#.#.#" +
	"##.#.#.#.#.#.#.#.#.#" +
	"##.b.#.b.#.b.#.b.#.#" +
	"##.a.#...#...#.a.#.#" +
	"###############+####" +
	"####################" + // 37
	"#...#ffffffffff#...#" +
	"#...#fdddffdddf#...#" +
	"#.a.#fd#dffd#df#.a.#" +
	"#...#dd#dddd#dd#...#" +
	"#>a.cd###dd###dc.a.+" +
	"#...#dd#dddd#dd#...#" +
	"#.a.#fd#dffd#df#.a.#" +
	"#...#fdddffdddf#...#" +
	"#...#ffffffffff#...#" +
	"#...############...#" +
	"####################" +
	"####################" + // 38
	"#..................#" +
	"#.......s..s.......#" +
	"#..r...c#cc#c...r..#" +
	"#.....c#....#c.....#" +
	"#>...c#..#c..#c....#" +
	"#...c#...#c...#c...#" +
	"#..c#c.c..#.c.c#c..#" +
	"#.c#..##cc#c##..#c.#" +
	"#.#...#..#...#...#c+" +
	"##....#..#...#....##" +
	"####################" +
	"####################" + // 39
	"#>.................#" +
	"#kkkkkk#..kkkkkkk#.#" +
	"#.kk#.kk#..kk#..k#.#" +
	"#.kk#.kk#..kk#..k#.#" +
	"#.kk#.kk#..kk#k#...#" +
	"#.kkkkk#...kkkk#...#" +
	"#.kk#......kk#k#...#" +
	"#.kk#......kk#.....#" +
	"#.kk#......kk#.....#" +
	"#kkk##....kkk##....+" +
	"####################" +
	"####################" + // 40
	"########ccccccd#####" +
	"#######ccscccdd#####" +
	"######ccccccddd#####" +
	"#####eeeeeeeddd#####" +
	"####>eeeeeeeddd#####" +
	"#####eeeeeeeddd#####" +
	"#####eee#eeedd######" +
	"#####eeeeeeed#######" +
	"#####eeeeeee########" +
	"###########+########" +
	"####################" +
	"####################" + // 41
	"#..................#" +
	"#g#g#g#g#..#g#g#g#g#" +
	"#g#g#g#g#..#g#g#g#g#" +
	"#g#g#g#g#..#g#g#g#g#" +
	"#g#g#g#g#h.#g#g#g#g#" +
	"#g#g#g#g#.o#g#g#g#g#" +
	"#g#g#g#g#..#g#g#g#g#" +
	"#g#g#g#g#..#g#g#g#g#" +
	"#g#g#g#g#..#g#g#g#g#" +
	"#.........>........#" +
	"##########+#########" +
	"####################" + // 42
	"#####ii######j+#####" +
	"####iiii#####ii#####" +
	"###.iiii.####ii#####" +
	"#####ii######ii#####" +
	"####eiihhhhhhiie####" +
	"####eiiooooooiie####" +
	"#####ii######ii#####" +
	"#####ii####.iiii.###" +
	"#####ii#####iiii####" +
	"#####j>######ii#####" +
	"####################" +
	"####################" + // 43
	"#...n...g#f...d....#" +
	"#..##n.#g#f#.d##...#" +
	"#..###n.g#f.d###...#" +
	"#..####ng#fd####...#" +
	"#>ccsccc.+.cccsccc.#" +
	"#..####df#gn####...#" +
	"#..###d.f#g.n###...#" +
	"#..##d.#f#g#.n##...#" +
	"#..#d.##f#g##.n#...#" +
	"#..d....f.g....n...#" +
	"####################" +
	"####################" + // 44
	"##.cc.#####.cc.#####" +
	"#.c##c##cc.c##c.sc.#" +
	"#c###.cc..c.##.c..c#" +
	"#c#.csc#.c#c#.c#c#.+" +
	"#>#c##c#c##c#c##c###" +
	"###c##c#c##c#c##c###" +
	"###c#c.#c##c#c##c###" +
	"###.c.##c#c.#c#c.###" +
	"########.c.##.c.####" +
	"####################" +
	"####################" +
	"####################" + // 45
	"#.ii.>..jj....ff...#" +
	"#i##i..j##j..f##f..#" +
	"#i##i..j##j..f##f..#" +
	"+.ii....jj....ff...#" +
	"##cc####cc####cc####" +
	"##cc####cc####cc####" +
	"#.ii....jj....dd...#" +
	"#i##i..j##j..d##d..#" +
	"#i##i..j##j..d##d..#" +
	"#.ii....jj....##...#" +
	"####################" +
	"####################" + // 46
	"#>.o.#ee#.h.h.h.#..#" +
	"#.o.o#eh#..o.o..#cg#" +
	"#..s.#ee....e......#" +
	"#o...#oee.h...h.####" +
	"#.o..####..o.o.....#" +
	"####.o.o.o.####o...#" +
	"#.c...o.o..hhh#oo..#" +
	"#gg#o..s....hh#ooo.#" +
	"+cc#.o..o..#hh#osoo#" +
	"#gg#o.o...h#hh#oooo#" +
	"####################" +
	"####################" + // 47
	"#....p.......p.....#" +
	"#.###p#l#r#l#p####.#" +
	"#.lllp#l#r#l#pllll.#" +
	"#.###p#l#r#l#p####.#" +
	"#.lllp#l#r#l#pllll.#" +
	"#>###p#l#r#l#p####.#" +
	"#.lllp#l#r#l#pllll.#" +
	"#.###p#l#r#l#p####.#" +
	"#.lllp#l#r#l#pllll.#" +
	"#..................#" +
	"#########+##########" +
	"####################" + // 48
	"#.....########.....#" +
	"#.f#d.#hhhhhh#.d#f.#" +
	"#.f#d.#c#ee#c#.d#f.#" +
	"#.f#d.#oo##oo#.d#f.#" +
	"#>###.iosggsoi.###.+" +
	"#.d#f.#oo##oo#.f#d.#" +
	"#.d#f.#c#ee#c#.f#d.#" +
	"#.d#f.#hhhhhh#.f#d.#" +
	"#.....########.....#" +
	"#........##........#" +
	"####################" +
	"####################" + // 49
	"#........>.........#" +
	"#.#c##cc#cc#cc##c#.#" +
	"#.#c.##c#cc#c##.c#.#" +
	"#.#rc.#cc##cc#.cr#.#" +
	"#..#rc.cs##sc.cr#..#" +
	"#...#rc.#cc#.cr#...#" +
	"#....#rc#cc#cr#....#" +
	"#.e...#llllll#...e.#" +
	"#..e...######...e..#" +
	"#........+#........#" +
	"####################" +
	"####################" + // 50
	"#abcdefghijklmnopqr#" +
	"#abcd####ij####opqr#" +
	"#abcd#fghij#..#opqr#" +
	"+abcd#fghij#..#opqr#" +
	"#>bcd####ij#..#opqr#" +
	"#abcdefg#ij#..#opqr#" +
	"#abcdefg#ij#..#opqr#" +
	"#abcd#fg#ij#..#opqr#" +
	"#abcd####ij####opqr#" +
	"#abcdefghijklmnopqr#" +
	"####################"

// levelPsw are the passwords of the levels, pswLen characters a level.
const levelPsw = "" +
	"INTROGOLEMKRONEMYDLOLEDENSTROPLILIERYBKAKLAUNBRAUN" + // 0-9
	"WHITEMOUSEDREAMIDEALILUZESAINTBELLSTROJAEMOCEBASTL" + // 10-19
	"ISLAMJEANSMETALMIKROSENZAGALAXSTONESLAVEFORTELUCIE" + // 20-29
	"NATURBIENEKATZEHYDROFAUNAULTRATODAYHORORGAPPARETAL" + // 30-39
	"CLOCKHOMERBUNNYFOTONQUACKHAFIKDIZZYSTARKZARAZDUKEN" + // 40-49
	"STORY" // 50

// solution are the solutions of the levels, a direction a step.
var solution = [levNum + 1]string{
	"RRRRRRRRRRRRRRRRRRRDLLLLLLLLLL",                            // 0
	"RRRRRRRRRRRRRRUUULLLLLLLLLLLDDDDDDRRRRRRRDRRRRRRRR",        // 1
	"UURRRRRRRRRULLLLLLURRRRRRULLLLLLURRRRRRRDDDDDDLLLLDD",      // 2
	"UUULURRULLURRULURRRRRRRRRRRRRDDDDDDLUUUULDDDDLUUUULDDDLLL", // 3
	// 4
	"RRRDDDDDRRRRRRRRRRULLLLURRULLLLDDLLLURRUURRRULLLLLLLLLUUUURRRRRR" +
		"RDRRRRRRRRULLLLLLLU",
	// 5
	"RRDDDRRRRULLLURRRULLLLURRRRULLLLLURRRRRRRRRRRRRRRDLDDDDDLUUUULDD" +
		"DDLUUUULDDDDDDLLLLLLLLLLLUUL",
	// 6
	"RRRRRDDDRRRRRULLLLURRRRULLLLURRRRULLLLURRRURRRRRDDDDDDDDDLLLLLLL" +
		"LLLLLLLL",
	// 7
	"RRRRDDLLDDRRRRRULURRRRDLDRRRRRRRUULLLULLLLLLLURRRRRRRRRRUULLLLLL" +
		"DLULDLULDLULLUURRRRRU",
	// 8
	"DDDDRRUUUUUUUURRRRRRDDDLLDDDDRRRRULLLURRRULLURULUURRRRRRDDDDDDDD" +
		"DLLLLLL",
	// 9
	"UUUUURDDDDDRRRRUURRRULLLURRRULLLURRRRUUULLLLLLLLDLULLLLLLLLDDDDR" +
		"RRRDLLLDRRRDLLDDRRRRRRD",
	// 10
	"DDDDDRRUUUUUUUUURRRRRRRRDLLLDRDLLLDDDDDDRRRRRRRRRURULURULURULURU" +
		"LULLLDDDDDDDDDLLLUUUUU",
	// 11
	"RRDDDLDRDRRRUUUUULLULURURRURRDRRRURRRRDDRDLDRDDDRDLDLLLUUUUULULL" +
		"LDDRRDLLDRRDLLLUUUUL",
	// 12
	"DDDDDRRUUUUURRRRRRDDDLLLLDDRURDRURDRURDRURDRUULLLLLUULLLLLLDDDLU" +
		"UUUUUURRRRRRRRRRRRRRRDDDLLDRRDDDRUUUUUUUULLLLLLLLLU",
	// 13
	"RRUULUURDRDDDLLLDDDDDRURURRURRURRURRULLLUURRDRRDRUUURRRRRDLLLDDD" +
		"RRRDDLLDDRRDLLLLUUULLLDRRDDLULLULDDLULDD",
	// 14
	"UUUURRDLDDDRRRUUUURRDDDDRRRULULURURRRDDRRRRRRDDDDLDRDDLLUUULLDDD" +
		"LUUULDDDLLLUURULLLLLLLLLUUL",
	// 15
	"RRUUUURRRRRRDDRDRDDDLDLLUURULULLLURRRRRRRRRDLLLDDDRRRDLLLLLLLLLU" +
		"RRRURULULLLURRRRUULLLLLLDDDDR",
	// 16
	"UUUUURRRDRDRDRDRRURURURURRDDDDDDLLLLLDRDDLLLLUURULLLLLUUUUUURRRR" +
		"DRRRRDLLLDRDD",
	// 17
	"DDDDRURDRUUULLUULUUUURDDDRDRUUULURRRRRRDDLLLDDRRRRRUURRDDDLDLDLL" +
		"ULLLDDDRURDRRRRUURDDD",
	// 18
	"DDDDRRRUUUURRRRDRDDDLLLURDRRRRRUULDDLLUUURUUULUUULLLDDRUURRDDDDL" +
		"LLLLLUUULDDDDDDDRUUURRRRRRRRRRRDDDDRRRUUUUUUUULDDDDDDLUUUULDLLLL" +
		"LUUUURRDRUR",
	// 19
	"DDDDDRUUUUURDDDDDRUUUUURDDDDDRUUUUURDDDDDRRRRRRRRRRRULLLLLLLLLLU" +
		"RRRRRRRRRRULLLLLLLLLLURRRRRRRRRRULLLLLLLLLURRRRRRRRRULLLLLLLLLUR" +
		"RRRRRRRRULLLLLLLLLLLLLLLLLDRRRRRRRDLLLLLLLDRRRRRRRD",
	// 20
	"RRRRRRDRRRURURRRDRDLLULDLDRRRRRDLLLLLLULLULULULDLLDDDDRRRDLLLLLU" +
		"UUUUUUURRRRRRRRRRRRRRRRRDDDDDDDDLUU",
	// 21
	"LDDRDDLLULULULULULLDDDDDRRULULUURRDRDRDRDRRRUUULLULULULULLLUURDR" +
		"URDRDRDRDRDRRRURURURURURRRDDLULDLDLDLDLDLLDDDRRRURURURURRDDLDDRR",
	// 22
	"DDDDDDRRRRUUURRDDDDDRRRUURUULLDRDDDRRRUUUUUUURDDRDDDDRDDRRULURUL" +
		"LURRULLURRULLLURRRULLLLLLLDRDRDDDDDDDLLLLLLUUUUUUURRUULDLULLLLLD" +
		"RRRRDLLLLDRRDLLDRRRDDLLLLDRRRDLLLL",
	// 23
	"DDDDDRRRRRRRUUULLLUURRDRRRDDLDDRRRUUUUUULULULULLLLLDLDLDDDDDDDRR" +
		"RRRRUUULLLUURDRRRRDLDDDRRRRRUUULUUUULULURRRDDDDDRRRRDLLLDRRRDLLD" +
		"RRR",
	// 24
	"DDLLULUULLLLLUURRRUUULLDRRDDLLLDDRRRRRDDRDRRRRURUURRRRRRUULLLLUU" +
		"URRRRDLLLLDDRRRRDDLLLLLLDDLDLLLLULUULLLLLDDDDRRRUULDLULUURRRRRDD" +
		"RDRRRRURUURRRRRRDDDDLLLLUURDRURRUULLLLLLDDLDLLUUURUUUUULLDRDLDRD" +
		"LDRDDLLLDDRRURRURRDDLLLD",
	// 25
	"RRRRRRRRRRRRRRRRRDLDDDDDDDDLUUUUUUULULLLLLLLLLLLLLDDDDDDDDRUUUUU" +
		"UURRDDDDDDDRUUUUUUURRRRRRRRDDDDDDDLUUUUUULLDDDDDDLUUUUUULLDDDDDD" +
		"D",
	// 26
	"LURURURURURURURURRRDRDRDRDRDRDRDRUUUUULULULULLLLLLLLLDDLLULLDRDL" +
		"DRRURRRURURRRRRDRDRDRDRDDDRDL",
	// 27
	"RDRDDLLDDRRRDDDDLLLUURRRUULLLUURRRRUUURRDDDDDLDDRRDRDLLUULUURUUU" +
		"RRURURRRRRRDDDDDRDLDLDLUUUULLDDDLUULLU",
	// 28
	"UUUURRRRRRRDRRRDDDRRDDRDRURULLURRULULDLDLLDLULDLLLDLDLULURRULLUR" +
		"URDRDRRUUUUR",
	// 29
	"DRRUUULDRDDLLLLLDDDDDLUURUUUUUUURRDDLLDDLLLLUUUULLDDRRDDDDDLLDDR" +
		"RRULUUUURRRRRRRRRRRRRUULUURRRDDLULDDDDDDRRDDLULDLUURUURRULLLDLLL" +
		"LLLLLLLLLLLURRRRRRRRRRRDDDDLLUU",
	// 30
	"RRRRRRUULULULUUURRDRDRDDDDDRRRUUUUURURURRDDDLDLDLDDRRRRRUUUUUUUU" +
		"ULLLLLLLLLLLLLLLDDDDDDDDDRRRRRUULLLUUUUURRRRDDDDDDDRRRUUUUUUURRR" +
		"RDDDDDLLLDDRRRRRRR",
	// 31
	"RRRRRRRRRRRRRRRRRDDDDLLLLLLLLLLLLLLLLLUUUUUURRRRRRRRRRRRRRRRRDDD" +
		"DDDLLLLLLLLLLLLLLLLLUUUURRRRRRRRRRRRRRRRRUUUULLLLLLLLLLLLLLLLLDD" +
		"DDDDRRRRRRRRRRRRRRRRRDDLLLLLLLLLD",
	// 32
	"UUUURDLDDDDDDRDRDRRRULULULULUURURURURRRDLDLDLDDRDRDRDRDRRRULULUL" +
		"ULUURURURURRRDLDLDLDDRDRDRDRDRRRULULULULUURURUURDRDDLDDRDR",
	// 33
	"UUUURDDRDDRRUURUURRRRRDRDDDDDDDLDLLLLLUULUULLDDLDDLUUUUURRRRRRUU" +
		"URURRRRRRRRRDDDDDLLLLDDDDLLLLLULUUULLLLDDLDDLUUURULURULUUURDDRDD" +
		"RRRRUUURURRRRRDDDDDRRDRDRDDRUUUUR",
	// 34
	"LLLLDRDRDRDRDRURURURURURRRRDDDDDDDDDLLUUUUUUUUULLLLDLLLLLDRRRRDL" +
		"LDDDLDLDLDRRURURURDRDRDLLULDLD",
	// 35
	"DDDDRRRRRRDRRUURURURULULULLDLDLDRDRUURRDLDDDDLLLLLLLLUUUUUUUUURR" +
		"RRRRRRRRRRRRRRRDDDDDDDDDLLLLLLLLLUULUUUUURU",
	// 36
	"DDDDDDDDDRRUUUUUUUUURRDDDDDDDDDRRUUUUUUUUURRDDDDDDDDDRRUUUUUUUUU" +
		"RRDDDDDDDDDRRUUUUUUUURRULLDDDDDDDDDLLUUUUUUUUULLDDDDDDDDDLLUUUUU" +
		"UUUULLDDDDDDDDDLLUUUUUUUUULLDDDDDDDDDLLURRUUUUUUURRDDDDDDDRRUUUU" +
		"UUURRDDDDDDDRRUUUUUUURRDDDDDDDRDD",
	// 37
	"RRRRURUURRDDRRRUURRDDRDRRUUUURRDLDRDLDRDLDRDLDRDLLUUUUULLULUULLD" +
		"DLLLUULLDDLDLLUUUULLDRDLDRDLDRDLDRDLDRRUUUUURRUUUURRRRRRRRRDDLUL" +
		"LLLDRDDLDLDRDLLLUULDDDRRRRRUUURDDDRURDRUULURURRRRR",
	// 38
	"UUUURRRRRRRRDDDLDDDDDDLURURULUUURRRDDDDDRDLLURUUURRDRDDDRRULULUL" +
		"ULULULLLDLDLDLDDDLLURURURRRUUURRDDRDRRULULULUULLLDLDLDLDLDLDLUUU" +
		"UURRUURRRRRRRDRURRRRRDDLULLLDRDRDRDRDRDRDR",
	// 39
	"RRRRRDDDDDLLLDDDDLLURUUUURULURULLURRRDRRDRDDLDDDDDRUUURDDDRUUURD" +
		"DDRRUUUURRUURRUULLLLLLULLLLLLLLLDDDDDDDDDRUUUUUUUURDDDDRRRDDDRRR" +
		"RRUUUUUURDDDRRDDDRRDRRR",
	// 40
	"RURURURURRRRRRDDDDDLDLDLLURURURULURULLLLDLDLDLDLDRRURURURURDDLDL" +
		"DDLLLLDRRRRRRD",
	// 41
	"RRUUUUUUUUULLLLLDDDDDDDDDRRRRRRRUUUUUUUUULLLLLLLLLDDDDDDDDDRRRRU" +
		"UURDDDRRRRRRUUUUUUUUULLLLLLDDDDDLUUUUULLLLLLDDDDDDDDDRRRRRRUUURD" +
		"DDRRRRRRRRUUUUUUUUULLLLLLLLDDDDDLUUUUULLLLLLLLDDDDDDDDDRRRRRRRRU" +
		"UURDDDD",
	"LURULURULLURUULURURDRDLDDRDRURDRURDRURDDDLDRDRURULUURULULURULUR", // 42
	// 43
	"DDDDDRRRURURURUUUULULULLLLDDDDDDDDDRRRRRRRRRRRRRRRRUUUUUUUUULLLL" +
		"DLDLDDDDRDRDRDLLLLLLUUUUUUUUULLLLLLLDDDDRRRRRRDDLDLDLDRRRRRRUUUU" +
		"RURRRRRUUUULLLLLLDDDDL",
	// 44
	"UUURURRRDDDLLDDDDRRURUUUURRURDDLDDDDDRRURUUUUULURURRRDRDLDLDDDDD" +
		"RRURUUUUUURRDDR",
	// 45
	"RRRRRRRRRRRRDDDLLDDDRRRDDDLLURULULLLDDDLLLLLLLLLLLLUUURUUURDDDRD" +
		"DRRRUURUUURDDDRDDRRRUURUUURRUUULLLDDLLLUULLLDDLLLUULLLDDDL",
	// 46
	"DDRDRDRDDRDRDRDRRUUURUULUURRDRRURRRRRUULDDLLLLDLLULLDDRDDDDLDLLU" +
		"LULULUUUUUULLLDRRDLLDDRRRDDRDRDRDRRURUUUULUURRDRRURRDDRRRDDDDDLL" +
		"LURRULLURRULLULLURUUULDDLUULDLULDDLLUULDDDRRRRRDLDDRRRDDDLUULLLU" +
		"ULLLLDRRRDLDRRRDLLLLLLURULULLLDRDDLUL",
	// 47
	"DDDDRRRRRRRRUUUUUUUUULLDDDDDDDDDRRRRUUUUUUUUULLLLLLLLLLDDDDDDDDD" +
		"RRRRRRRRRRRRUUUUUUUUULLLLLLLLLLLLDDDDDDDDDRRRRRRRRRRRRURDRURDRUR" +
		"UUUUUUUULLLLLLLLLLLLLDDDDDDDDDRRRRRRRRURDRURDRURUUUUUUUULLLLLLLL" +
		"LLLLLLLLLDDDDDDDDDRRRRRRRRRRRRRRRRRULLLLLUURRRRRUULLLLLUURRRRRUU" +
		"LLLLLLLLLLLLLLLLLDDRRRRDDLLLLDDRRRRDDLLLLDRRRRRRRRD",
	// 48
	"UUUURRRRDDDDRRRRRRRRRURUUURRDDDRDDDDDLLLLUUUULLDLULLLULDLLULUUUL" +
		"LDDDLDDRDDLDDRURDRRUULUURURRUUURRDRURRDDLDLLLDLDDRRURDRRUUURRDRD" +
		"DDRRUUURUR",
	// 49
	"LLDDDDDDDRRUURDDRRUUUUUUURRRRRRDDDDDDDDDLLLLLURRRRULLURRUURUUUUL" +
		"LDDDLDLDLDLUULURUURULLLDDLUULLLDRDDRDLDDLULULULUUULLDDDDDDRDDRDR" +
		"RRRRR",
	// 50
	"RURRDLDRRRRDDLULLLLULDDDDRUURRDLDRRRRRRUUUUUULLLURRRURDDDDDDDDRR" +
		"RRRRRRULLLURRRULLLURRRULLLURRRULLLURRRULLLLLLLLLLLLLLLLLDRRRDLLL" +
		"DL",
}
