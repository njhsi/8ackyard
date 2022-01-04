package backyard

type IndexJob struct {
	FileName string
	Related  RelatedFiles
	IndexOpt IndexOptions
	Ind      *Index
}

func IndexWorker(jobs <-chan IndexJob) {
	for job := range jobs {
		log.Infof("IndexWorker:                           %s", job.FileName)
		IndexRelated(job.Related, job.Ind, job.IndexOpt)
	}
}
